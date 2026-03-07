//go:build wireinject
// +build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"github.com/KATOmemorial/cronyx/internal/biz"
	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/config"
	"github.com/KATOmemorial/cronyx/internal/data"
	"github.com/KATOmemorial/cronyx/internal/discovery"
	"github.com/KATOmemorial/cronyx/internal/server"
	"github.com/KATOmemorial/cronyx/internal/service"
)

// App 定义一个结构体来包裹我们需要的所有组件
type App struct {
	Engine *gin.Engine
	Master *discovery.Master
}

// NewApp 构造函数
func NewApp(engine *gin.Engine, master *discovery.Master) *App {
	return &App{
		Engine: engine,
		Master: master,
	}
}

// initApp 初始化应用，现在只返回一个 *App 主对象
func initApp() (*App, func(), error) {
	panic(wire.Build(
		config.ProviderSet,
		common.ProviderSet,
		data.ProviderSet,
		discovery.MasterProviderSet,
		biz.ProviderSet,
		service.ProviderSet,
		server.ProviderSet,
		NewApp, // 👈 告诉 Wire 怎么组装 App
	))
}
