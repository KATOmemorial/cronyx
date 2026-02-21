package server

import (
	"context"
	"fmt"
	"net"

	"github.com/google/wire"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/KATOmemorial/cronyx/api/proto"
	"github.com/KATOmemorial/cronyx/internal/biz"
	"github.com/KATOmemorial/cronyx/internal/config"
)

// GrpcProviderSet 专门给 Worker 用
var GrpcProviderSet = wire.NewSet(NewWorkerGrpcServer)

type WorkerGrpcServer struct {
	proto.UnimplementedWorkerServiceServer
	exec *biz.Executor
	log  *zap.Logger
	conf *config.Config
}

func NewWorkerGrpcServer(exec *biz.Executor, logger *zap.Logger, conf *config.Config) *WorkerGrpcServer {
	return &WorkerGrpcServer{
		exec: exec,
		log:  logger,
		conf: conf,
	}
}

// StopTask 实现 gRPC 接口
func (s *WorkerGrpcServer) StopTask(ctx context.Context, req *proto.StopRequest) (*proto.StopReply, error) {
	s.log.Info("🔪 Received Kill Request", zap.String("target", req.TaskId))

	count := s.exec.KillTask(req.TaskId)

	if count == 0 {
		return &proto.StopReply{Success: false, Message: "No matching task found"}, nil
	}
	return &proto.StopReply{Success: true, Message: fmt.Sprintf("Killed %d tasks", count)}, nil
}

// Start 启动 gRPC 服务 (非阻塞，内部使用 goroutine)
func (s *WorkerGrpcServer) Start() {
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.conf.Server.GrpcPort))
		if err != nil {
			s.log.Fatal("Failed to listen gRPC", zap.Error(err))
		}

		grpcServer := grpc.NewServer()
		proto.RegisterWorkerServiceServer(grpcServer, s)

		s.log.Info("🚀 gRPC Server started", zap.Int("port", s.conf.Server.GrpcPort))
		if err := grpcServer.Serve(lis); err != nil {
			s.log.Fatal("Failed to serve gRPC", zap.Error(err))
		}
	}()
}
