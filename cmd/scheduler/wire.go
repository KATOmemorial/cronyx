//go:build wireinject
// +build wireinject

package main

import (
	"github.com/IBM/sarama"
	"github.com/google/wire"
	"go.uber.org/zap"

	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/config"
	"github.com/KATOmemorial/cronyx/internal/data"
	"github.com/KATOmemorial/cronyx/internal/discovery" // 👈 新增导入
)

// App 调度器应用结构体
type App struct {
	conf     *config.Config
	logger   *zap.Logger
	data     *data.Data
	producer sarama.SyncProducer
	election *discovery.Election // 👈 新增依赖
}

// NewApp 构造函数
func NewApp(conf *config.Config, logger *zap.Logger, data *data.Data, producer sarama.SyncProducer, election *discovery.Election) *App {
	return &App{
		conf:     conf,
		logger:   logger,
		data:     data,
		producer: producer,
		election: election, // 👈 赋值
	}
}

// initApp 初始化依赖
func initApp() (*App, func(), error) {
	panic(wire.Build(
		config.ProviderSet,
		common.ProviderSet,
		data.ProviderSet,
		discovery.ElectionProviderSet, // 👈 告诉 Wire 怎么创建 Election
		NewApp,
	))
}
