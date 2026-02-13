package main

import (
	"time"

	"go.uber.org/zap" // 👈 必须引入 zap

	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/config"
	"github.com/KATOmemorial/cronyx/internal/discovery"
)

func main() {
	// 1. 初始化
	config.LoadConfig("./configs/config.yaml")
	common.InitLogger()

	// 2. 启动 Master (监听者)
	master := discovery.NewMaster()
	// 启动监听协程
	master.WatchWorkers()

	common.Log.Info("👀 Master is watching... Start/Stop your Worker now!")

	// 3. 模拟主程序运行，每 3 秒打印一次当前的 Worker 列表
	for {
		time.Sleep(3 * time.Second)

		// 获取当前活着的节点列表
		workers := master.GetWorkers()

		// 👇 修正点：使用 zap.Int 和 zap.Any 包裹参数
		common.Log.Info("📊 Current Active Workers",
			zap.Int("count", len(workers)),
			zap.Any("nodes", workers), // zap.Any 可以打印 map
		)
	}
}
