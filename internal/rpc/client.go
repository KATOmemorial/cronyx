package rpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/KATOmemorial/cronyx/api/proto"
	"github.com/KATOmemorial/cronyx/internal/common"
	"go.uber.org/zap"
)

// KillTask 远程调用 Worker 强杀任务
// targetIP: 例如 "192.168.1.5:9090"
// taskID: 任务的唯一 ID (RunID)
func KillTask(targetIP, taskID string) error {
	// 1. 建立连接 (不使用 TLS，因为是内网通信)
	conn, err := grpc.Dial(targetIP, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to worker %s: %v", targetIP, err)
	}
	defer conn.Close()

	// 2. 创建客户端
	client := proto.NewWorkerServiceClient(conn)

	// 3. 发起调用 (设置 3 秒超时)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := client.StopTask(ctx, &proto.StopRequest{TaskId: taskID})
	if err != nil {
		return fmt.Errorf("rpc call failed: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("worker returned error: %s", resp.Message)
	}

	common.Log.Info("Kill command sent successfully",
		zap.String("target", targetIP),
		zap.String("task_id", taskID),
	)
	return nil
}
