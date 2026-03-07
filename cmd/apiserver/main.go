package main

import (
	"fmt"

	"github.com/KATOmemorial/cronyx/internal/config"
)

func main() {
	// app 现在是 *App 结构体
	app, cleanup, err := initApp()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// 开启后台协程，实时监听 Etcd 中 Worker 节点的上下线！
	app.Master.WatchWorkers()

	conf := config.NewConfig()
	addr := fmt.Sprintf(":%d", conf.Server.HttpPort)
	fmt.Printf("🚀 API Server starting on %s\n", addr)

	// 启动 Http 服务
	if err := app.Engine.Run(addr); err != nil {
		panic(err)
	}
}
