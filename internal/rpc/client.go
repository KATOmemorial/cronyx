package rpc

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/KATOmemorial/cronyx/api/proto"
)

// KillTask 远程调用 Worker 强杀任务
func KillTask(targetIP, taskID string, logger *zap.Logger) error {
	// 1. 建立连接 (不使用 TLS，因为是内网通信) [cite: 39]
	conn, err := grpc.Dial(targetIP, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to worker %s: %v", targetIP, err)
	}
	defer conn.Close()

	// 2. 创建客户端 [cite: 39]
	client := proto.NewWorkerServiceClient(conn)

	// 3. 发起调用 (设置 3 秒超时) [cite: 39]
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := client.StopTask(ctx, &proto.StopRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("rpc call failed: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("worker returned: %s", resp.Message)
	}

	logger.Info("🔪 Kill command executed successfully",
		zap.String("target", targetIP),
		zap.String("task_id", taskID),
	)
	return nil
}
