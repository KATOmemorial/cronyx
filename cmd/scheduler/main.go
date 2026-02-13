package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/config"
	"github.com/KATOmemorial/cronyx/internal/model"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func main() {
	// 1. 加载配置
	config.LoadConfig("./configs/config.yaml")

	// 2. 初始化日志
	common.InitLogger()

	// 3. 初始化 DB & Redis
	common.InitDB()
	common.InitRedis()

	// 4. Kafka 配置 (从配置读取)
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	// 使用配置里的 Brokers
	producer, err := sarama.NewSyncProducer(config.AppConfig.Kafka.Brokers, saramaConfig)
	if err != nil {
		common.Log.Fatal("Failed to start Kafka producer", zap.Error(err))
	}
	defer producer.Close()

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	common.Log.Info("Distributed Scheduler started!", zap.String("env", config.AppConfig.System.Env))

	// 2. 调度主循环
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		var jobs []model.JobInfo
		now := time.Now()

		// A. 扫描任务
		if err := common.DB.Where("status = ? AND next_time <= ?", 1, now.Unix()).Find(&jobs).Error; err != nil {
			common.Log.Error("Failed to fetch jobs", zap.Error(err))
			continue
		}

		// B. 遍历处理 (带分布式锁)
		for _, job := range jobs {
			// --- 抢锁开始 ---

			// 锁的 Key：cronyx:lock:任务ID:本次计划执行时间
			// 这样设计是为了保证：同一个任务的同一个执行周期，只能被锁一次
			lockKey := fmt.Sprintf("cronyx:lock:%d:%d", job.ID, job.NextTime)

			// SetNX (Set if Not Exists)
			// 参数：Context, Key, Value, Expiration
			// 锁有效期设为 5 秒 (防止死锁，5秒后自动释放)
			acquired, err := common.RDB.SetNX(context.Background(), lockKey, 1, 5*time.Second).Result()
			if err != nil {
				common.Log.Error("Redis lock Error", zap.Uint("job_id", job.ID), zap.Error(err))
				continue
			}

			if !acquired {
				// 没抢到锁，说明别的节点正在处理，我直接跳过
				common.Log.Info(" Job locked by another node, skipping...\n", zap.Uint("job_id", job.ID), zap.Int64("next_time", job.NextTime))
				continue
			}

			// --- 锁成功，开始干活 ---

			common.Log.Info("   Lock acquired for Job, executing...", zap.Uint("job_id", job.ID), zap.String("name", job.Name))

			// 1. 发送 Kafka
			taskID := fmt.Sprintf("%d-%d", job.ID, now.Unix())
			event := common.TaskEvent{
				TaskID:    taskID,
				Command:   job.Command,
				Timestamp: now.Unix(),
			}
			bytes, _ := json.Marshal(event)
			msg := &sarama.ProducerMessage{
				Topic: config.AppConfig.Kafka.Topic,
				Value: sarama.ByteEncoder(bytes),
			}
			_, _, err = producer.SendMessage(msg)
			if err != nil {
				common.Log.Error("Failed to send to kafka", zap.Uint("job_id", job.ID), zap.Error(err))
				continue
			}

			// 2. 计算下次时间并更新 DB
			schedule, err := parser.Parse(job.CronExpr)
			if err != nil {
				common.Log.Error("Invalid CronExpr", zap.Uint("job_id", job.ID), zap.Error(err))
				continue
			}
			nextTime := schedule.Next(now)
			common.DB.Model(&job).Update("next_time", nextTime.Unix())
			common.Log.Info("Job rescheduled", zap.Uint("job_id", job.ID), zap.Time("next_run", nextTime))
		}
	}
}
