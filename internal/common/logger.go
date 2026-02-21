package common

import (
	"os"

	"github.com/google/wire"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/KATOmemorial/cronyx/internal/config"
)

// ProviderSet 导出给 Wire 使用
// Wire 会看到这个 Set，知道："哦，如果有人需要 *zap.Logger，我就调用 NewLogger 来创建。"
var ProviderSet = wire.NewSet(NewLogger)

// Log 为了兼容旧代码，我们暂时保留这个全局变量
// 但在新的依赖注入链中，我们尽量使用 NewLogger 返回的对象
var Log *zap.Logger

// NewLogger 初始化日志 (构造函数模式)
// 注意：这里我们让它接收 *config.Config，Wire 会自动把 Config 注入进来！
func NewLogger(conf *config.Config) *zap.Logger {
	// 1. 定义日志级别
	// 如果是 dev 环境，打印 Debug 级别；否则打印 Info 级别
	var level zapcore.Level
	if conf.System.Env == "dev" {
		level = zap.DebugLevel
	} else {
		level = zap.InfoLevel
	}

	// 2. 配置编码器 (JSON 格式，适合机器读，也适合 ELK 收集)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // 时间格式: 2026-02-07T14:00:00.000Z

	// 3. 核心配置
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig), // 编码器
		zapcore.AddSync(os.Stdout),            // 输出到控制台
		level,                                 // 日志级别 (动态)
	)

	// 4. 创建 Logger
	// AddCaller: 打印调用行号
	logger := zap.New(core, zap.AddCaller())

	// 5. 替换全局 Logger (兼容旧代码)
	// 这样即使某些老地方还在用 zap.L() 或 common.Log，也能用到新配置
	zap.ReplaceGlobals(logger)
	Log = logger

	logger.Info("Zap Logger initialized",
		zap.String("env", conf.System.Env),
		zap.String("level", level.String()),
	)

	return logger
}
