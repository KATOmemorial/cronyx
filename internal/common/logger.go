package common

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 全局 Logger
var Log *zap.Logger

// InitLogger 初始化日志
func InitLogger() {
	// 配置编码器 (JSON 格式，适合机器读)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // 时间格式: 2026-02-07T14:00:00.000Z

	// 核心配置
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig), // 编码器
		zapcore.AddSync(os.Stdout),            // 输出到控制台
		zap.InfoLevel,                         // 日志级别
	)

	// 开启开发模式堆栈跟踪
	Log = zap.New(core, zap.AddCaller())

	// 替换全局的 logger (可选)
	zap.ReplaceGlobals(Log)

	Log.Info("Zap Logger initialized")
}
