package server

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"

	"github.com/KATOmemorial/cronyx/internal/config"
	"github.com/KATOmemorial/cronyx/internal/service"
)

// ProviderSet 导出
var ProviderSet = wire.NewSet(NewHTTPServer)

// NewHTTPServer 初始化 Gin 引擎并注册路由
// Wire 会自动注入 conf 和 jobService
func NewHTTPServer(conf *config.Config, job *service.JobService) *gin.Engine {
	// 根据配置设置 Gin 模式
	if conf.System.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// 注册路由
	v1 := r.Group("/api/v1")
	{
		// 注意：这里 job.CreateHandler 必须在 internal/service/job.go 里定义并公开
		v1.POST("/job", job.CreateHandler)
		v1.GET("/jobs", job.ListHandler)
		v1.POST("/job/kill", job.KillHandler)
	}

	return r
}
