package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/KATOmemorial/cronyx/api/proto"
	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/config"
	"github.com/KATOmemorial/cronyx/internal/discovery"
	"github.com/KATOmemorial/cronyx/internal/model"
)

// --- 全局任务管理器 ---
// 用来存储正在运行的任务，以便 Kill 掉它们
var (
	taskMap  = make(map[string]context.CancelFunc)
	taskLock sync.Mutex
)

// --- gRPC 服务实现 ---
type WorkerServer struct {
	proto.UnimplementedWorkerServiceServer
}

// StopTask 实现 gRPC 接口：强杀任务
func (s *WorkerServer) StopTask(ctx context.Context, req *proto.StopRequest) (*proto.StopReply, error) {
	targetID := req.TaskId // 可能是 "101-123456" (精确) 或 "101" (模糊)
	common.Log.Info("🔪 Received Kill Request", zap.String("target", targetID))

	killedCount := 0

	taskLock.Lock()
	defer taskLock.Unlock()

	for taskID, cancel := range taskMap {
		// 逻辑：如果 taskMap 里的 Key 包含了 targetID，就杀掉
		// 例如：正在跑 "101-17000"，目标是 "101"，匹配成功！
		if strings.HasPrefix(taskID, targetID) {
			cancel()                // 杀！
			delete(taskMap, taskID) // 移除
			killedCount++
			common.Log.Warn("💀 Task killed", zap.String("task_id", taskID))
		}
	}

	if killedCount == 0 {
		return &proto.StopReply{Success: false, Message: "No matching task found"}, nil
	}

	return &proto.StopReply{Success: true, Message: fmt.Sprintf("Killed %d tasks", killedCount)}, nil
}

func main() {
	// 1. 初始化
	config.LoadConfig("./configs/config.yaml")
	common.InitLogger()
	common.InitDB()

	// 2. 服务注册
	ip, err := common.GetOutboundIP()
	if err != nil {
		common.Log.Fatal("Failed to get local IP", zap.Error(err))
	}

	// gRPC 端口
	grpcPort := config.AppConfig.Server.GrpcPort
	addr := fmt.Sprintf("%s:%d", ip, grpcPort)

	register := discovery.NewServiceRegister()
	err = register.Register("/cronyx/worker/"+addr, addr, 10)
	if err != nil {
		common.Log.Fatal("Failed to register to Etcd", zap.Error(err))
	}
	defer register.Close()
	common.Log.Info("Worker registered", zap.String("addr", addr))

	// --- 3. 启动 gRPC Server (新增) ---
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
		if err != nil {
			common.Log.Fatal("Failed to listen gRPC", zap.Error(err))
		}

		s := grpc.NewServer()
		// 注册服务
		proto.RegisterWorkerServiceServer(s, &WorkerServer{})

		common.Log.Info("gRPC Server started", zap.Int("port", grpcPort))
		if err := s.Serve(lis); err != nil {
			common.Log.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()

	// 4. 启动 Kafka 消费者
	saramaConfig := sarama.NewConfig()
	consumer, err := sarama.NewConsumer(config.AppConfig.Kafka.Brokers, saramaConfig)
	if err != nil {
		common.Log.Fatal("Failed to start Kafka consumer", zap.Error(err))
	}
	defer consumer.Close()

	partitionConsumer, err := consumer.ConsumePartition(config.AppConfig.Kafka.Topic, 0, sarama.OffsetNewest)
	if err != nil {
		common.Log.Fatal("Failed to consume partition", zap.Error(err))
	}
	defer partitionConsumer.Close()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 5. 消费循环
	go func() {
		for msg := range partitionConsumer.Messages() {
			var event common.TaskEvent
			json.Unmarshal(msg.Value, &event)

			common.Log.Info("⚡ Executing Job", zap.String("task_id", event.TaskID))

			// --- 核心：创建可取消的 Context ---
			// 如果收到 gRPC 的 cancel()，这个 ctx.Done() 就会关闭
			ctx, cancel := context.WithCancel(context.Background())

			// 存入 Map
			taskLock.Lock()
			taskMap[event.TaskID] = cancel
			taskLock.Unlock()

			// --- 执行命令 (使用 CommandContext) ---
			// 这种方式启动的命令，一旦 ctx 被 cancel，进程会被自动 Kill
			startTime := time.Now()
			cmd := exec.CommandContext(ctx, "/bin/sh", "-c", event.Command)
			output, err := cmd.CombinedOutput()
			endTime := time.Now()

			// 执行完（或者被Kill后），从 Map 清理掉
			taskLock.Lock()
			delete(taskMap, event.TaskID)
			taskLock.Unlock()

			// 判断是被 Kill 的还是自然失败的
			status := 1
			errMsg := ""
			if err != nil {
				status = 0
				// 如果是 context canceled，说明是被强杀的
				if ctx.Err() == context.Canceled {
					errMsg = "Task killed by user"
					common.Log.Warn("Task killed successfully", zap.String("task_id", event.TaskID))
				} else {
					errMsg = err.Error()
					common.Log.Error("Execution failed", zap.Error(err))
				}
			} else {
				common.Log.Info("Execution success")
			}

			// JobID 解析逻辑 (简化)
			var jobID int
			parts := strings.Split(event.TaskID, "-")
			if len(parts) > 0 {
				jobID, _ = strconv.Atoi(parts[0])
			}

			// 入库
			jobLog := model.JobLog{
				JobID:     uint(jobID),
				Command:   event.Command,
				Output:    string(output),
				Error:     errMsg,
				PlanTime:  event.Timestamp,
				StartTime: startTime.UnixMilli(),
				EndTime:   endTime.UnixMilli(),
				Status:    status,
			}
			common.DB.Create(&jobLog)
		}
	}()

	common.Log.Info("Worker is running...")
	<-sigChan
	common.Log.Warn("Worker shutting down...")
}
