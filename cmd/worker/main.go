package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"

	"github.com/KATOmemorial/cronyx/internal/common"
)

// ConsumerHandler 实现 sarama.ConsumerGroupHandler 接口
type ConsumerHandler struct {
	app  *App
	pool *ants.Pool
}

func (h *ConsumerHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *ConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

// ConsumeClaim 核心消费逻辑
func (h *ConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		m := msg // 拷贝，防止闭包坑

		err := h.pool.Submit(func() {
			var event common.TaskEvent
			json.Unmarshal(m.Value, &event)

			h.app.logger.Info("⚡ Executing Job", zap.String("task_id", event.TaskID))

			// 执行任务
			output, err := h.app.executor.StartExecution(context.Background(), event.TaskID, event.Command)

			status := 1
			errMsg := ""
			if err != nil {
				status = 0
				errMsg = err.Error()
			}

			// 解析 JobID
			var jobID int
			parts := strings.Split(event.TaskID, "-")
			if len(parts) > 0 {
				jobID, _ = strconv.Atoi(parts[0])
			}

			h.app.logger.Info("📊 Job Result",
				zap.Int("job_id", jobID),
				zap.String("output", output),
				zap.String("error", errMsg),
				zap.Int("status", status),
			)

			// 🔥 必须标记消息已消费，否则下次重启还会再次消费！
			session.MarkMessage(m, "")
		})

		if err != nil {
			h.app.logger.Error("Failed to submit to ants pool", zap.Error(err))
		}
	}
	return nil
}

func (app *App) Run() {
	app.grpcServer.Start()

	ip, err := common.GetOutboundIP()
	if err != nil {
		app.logger.Fatal("Failed to get local IP", zap.Error(err))
	}
	addr := fmt.Sprintf("%s:%d", ip, app.conf.Server.GrpcPort)

	err = app.registrar.Register("/cronyx/worker/"+addr, addr, 10)
	if err != nil {
		app.logger.Fatal("Failed to register to Etcd", zap.Error(err))
	}
	defer app.registrar.Close()
	app.logger.Info("👷 Worker registered", zap.String("addr", addr))

	pool, err := ants.NewPool(100)
	if err != nil {
		app.logger.Fatal("Failed to init ants pool", zap.Error(err))
	}
	defer pool.Release()

	// 初始化 Handler
	handler := &ConsumerHandler{
		app:  app,
		pool: pool,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动消费者组消费 (它是一个死循环，需要放在 goroutine 里)
	go func() {
		for {
			// `Consume` 应该在一个无限循环中被调用，因为当服务器端 rebalance 时，
			// 这个函数会返回并需要被重新调用来获取新的 claims。
			if err := app.consumerGroup.Consume(ctx, []string{app.conf.Kafka.Topic}, handler); err != nil {
				app.logger.Error("Error from consumer", zap.Error(err))
			}
			// 检查 ctx 是否被取消，若是则退出循环
			if ctx.Err() != nil {
				return
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	app.logger.Info("✅ Worker is running with Consumer Group...")
	<-sigChan
	app.logger.Warn("🛑 Worker shutting down...")
}

func main() {
	app, cleanup, err := initApp()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	app.Run()
}
