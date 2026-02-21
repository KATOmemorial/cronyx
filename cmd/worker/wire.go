//go:build wireinject
// +build wireinject

package main

import (
	"github.com/IBM/sarama"
	"github.com/google/wire"
	"go.uber.org/zap"

	"github.com/KATOmemorial/cronyx/internal/biz"
	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/config"
	"github.com/KATOmemorial/cronyx/internal/data"
	"github.com/KATOmemorial/cronyx/internal/discovery"
	"github.com/KATOmemorial/cronyx/internal/server"
)

// App Worker 应用结构
type App struct {
	conf          *config.Config
	logger        *zap.Logger
	consumerGroup sarama.ConsumerGroup // 👈 这里改名并改类型了
	registrar     *discovery.ServiceRegister
	executor      *biz.Executor
	grpcServer    *server.WorkerGrpcServer
	repo          biz.JobRepo
}

func NewApp(
	conf *config.Config,
	logger *zap.Logger,
	consumerGroup sarama.ConsumerGroup, // 👈 这里也改
	registrar *discovery.ServiceRegister,
	executor *biz.Executor,
	grpcServer *server.WorkerGrpcServer,
	repo biz.JobRepo,
) *App {
	return &App{
		conf:          conf,
		logger:        logger,
		consumerGroup: consumerGroup, // 👈 赋值对应修改
		registrar:     registrar,
		executor:      executor,
		grpcServer:    grpcServer,
		repo:          repo,
	}
}

// ... 下面的 wire.Build 逻辑不用改，Wire 会自动适配！

// ProviderSet 定义 Discovery 相关的注入
// 因为 discovery 包还没把 NewServiceRegister 加入 ProviderSet，我们这里手动组装
var DiscoverySet = wire.NewSet(discovery.NewServiceRegister)

func initApp() (*App, func(), error) {
	panic(wire.Build(
		config.ProviderSet,
		common.ProviderSet,
		data.ProviderSet,
		biz.NewExecutor,        // 注入 Executor
		server.GrpcProviderSet, // 注入 gRPC Server
		DiscoverySet,           // 注入 ServiceRegister
		NewApp,
	))
}
