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
	"time" // 👈 新增引入 time 包

	"github.com/IBM/sarama"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"

	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/model" // 👈 新增引入 model 包
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

			// 记录执行开始时间 (毫秒)
			startTime := time.Now().UnixMilli()

			// 执行任务
			output, err := h.app.executor.StartExecution(context.Background(), event.TaskID, event.Command)

			// 记录执行结束时间 (毫秒)
			endTime := time.Now().UnixMilli()

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

			// --- 👇 核心改造：组装日志对象并写入 MySQL ---
			jobLog := &model.JobLog{
				JobID:     uint(jobID),
				Command:   event.Command,
				Output:    output,
				Error:     errMsg,
				PlanTime:  event.Timestamp * 1000, // Scheduler 传过来的是秒级时间戳，转为毫秒
				RealTime:  event.Timestamp * 1000, // 简单起见，实际调度时间暂与计划时间一致
				StartTime: startTime,
				EndTime:   endTime,
				Status:    status,
			}

			// 调用我们之前在 repo 中写好的 CreateLog 方法
			if dbErr := h.app.repo.CreateLog(context.Background(), jobLog); dbErr != nil {
				h.app.logger.Error("Failed to save job log", zap.Error(dbErr))
			} else {
				h.app.logger.Info("💾 Job log saved to database", zap.Uint("job_id", jobLog.JobID))
			}
			// --- 👆 核心改造结束 ---

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
	addr := fmt.Sprintf(":%d", ip, app.conf.Server.GrpcPort)

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

	// 启动消费者组消费
	go func() {
		for {
			if err := app.consumerGroup.Consume(ctx, []string{app.conf.Kafka.Topic}, handler); err != nil {
				app.logger.Error("Error from consumer", zap.Error(err))
			}
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
