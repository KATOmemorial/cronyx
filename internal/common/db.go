package common

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/KATOmemorial/cronyx/internal/config"
	"github.com/KATOmemorial/cronyx/internal/model"
)

var (
	DB  *gorm.DB
	RDB *redis.Client
)

// InitDB 初始化 MySQL (读取配置)
func InitDB() {
	// 从配置中读取 DSN
	dsn := config.AppConfig.MySQL.DSN

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		// 使用新的 Logger
		Log.Fatal("Failed to connect to MySQL", zap.Error(err))
	}

	// 自动迁移
	err = DB.AutoMigrate(&model.JobInfo{}, &model.JobLog{})
	if err != nil {
		Log.Fatal("Failed to migrate database", zap.Error(err))
	}

	sqlDB, _ := DB.DB()
	//  从配置中读取连接池设置
	sqlDB.SetMaxIdleConns(config.AppConfig.MySQL.MaxIdle)
	sqlDB.SetMaxOpenConns(config.AppConfig.MySQL.MaxOpen)
	sqlDB.SetConnMaxLifetime(time.Hour)

	Log.Info("MySQL Connected & Migrated successfully!")
}

// InitRedis 初始化 Redis (读取配置)
func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		// 从配置中读取 Redis 地址
		Addr:     config.AppConfig.Redis.Addr,
		Password: config.AppConfig.Redis.Password,
		DB:       config.AppConfig.Redis.DB,
	})

	ctx := context.Background()
	_, err := RDB.Ping(ctx).Result()
	if err != nil {
		Log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	Log.Info("Redis Connected successfully!")
}
