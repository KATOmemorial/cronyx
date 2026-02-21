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
	"github.com/KATOmemorial/cronyx/internal/server"
	"github.com/KATOmemorial/cronyx/internal/service"
)

// initApp 初始化整个应用
// 返回 gin.Engine (用于启动 HTTP 服务) 和 cleanup 函数 (用于关闭 DB/Redis)
func initApp() (*gin.Engine, func(), error) {
	panic(wire.Build(
		config.ProviderSet,  // 1. Config
		common.ProviderSet,  // 2. Logger
		data.ProviderSet,    // 3. Data (DB/Redis) & Repo
		biz.ProviderSet,     // 4. Biz (UseCase)
		service.ProviderSet, // 5. Service (HTTP Handlers)
		server.ProviderSet,  // 6. Server (Gin Engine)
	))
}
