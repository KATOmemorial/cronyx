package main

import (
	"fmt"

	"github.com/KATOmemorial/cronyx/internal/config"
)

func main() {
	// 1. 依赖注入初始化 (调用 wire 生成的代码)
	// app 就是 *gin.Engine
	app, cleanup, err := initApp()
	if err != nil {
		panic(err)
	}
	// 确保程序退出时关闭数据库连接
	defer cleanup()

	// 2. 为了获取端口号，我们还得手动加载一下配置
	// (或者你也可以让 initApp 返回 *config.Config)
	conf := config.NewConfig()

	// 3. 启动服务
	addr := fmt.Sprintf(":%d", conf.Server.HttpPort)
	fmt.Printf("🚀 API Server starting on %s\n", addr)

	if err := app.Run(addr); err != nil {
		panic(err)
	}
}
