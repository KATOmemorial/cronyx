package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/model"
)

// Run 启动调度器主循环
func (app *App) Run() {
	app.logger.Info("🚀 Distributed Scheduler started", zap.String("env", app.conf.System.Env))

	// 1. 启动后台竞选 Leader
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ip, _ := common.GetOutboundIP()
	nodeVal := fmt.Sprintf("%s-%d", ip, time.Now().UnixNano())

	// "/cronyx/election/scheduler" 是所有调度器竞选的同一个“王座”
	app.election.Campaign(ctx, "/cronyx/election/scheduler", nodeVal)

	// 2. 调度主循环
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 🔥 核心逻辑：如果我不是 Leader，我就什么都不干，直接跳过！
		if !app.election.IsLeader() {
			continue
		}

		// --- 下面只有 Leader 才会执行 ---
		var jobs []model.JobInfo
		now := time.Now()

		// A. 扫描任务
		if err := app.data.DB.Where("status = ? AND next_time <= ?", 1, now.Unix()).Find(&jobs).Error; err != nil {
			app.logger.Error("Failed to fetch jobs", zap.Error(err))
			continue
		}

		// B. 遍历处理 (不需要 Redis 锁了！)
		for _, job := range jobs {
			app.logger.Info("📅 Scheduling job", zap.Uint("job_id", job.ID), zap.String("name", job.Name))

			// 发送 Kafka
			taskID := fmt.Sprintf("%d-%d", job.ID, now.Unix())
			event := common.TaskEvent{
				TaskID:    taskID,
				Command:   job.Command,
				Timestamp: now.Unix(),
			}
			bytes, _ := json.Marshal(event)

			msg := &sarama.ProducerMessage{
				Topic: app.conf.Kafka.Topic,
				Value: sarama.ByteEncoder(bytes),
			}

			if _, _, err := app.producer.SendMessage(msg); err != nil {
				app.logger.Error("Failed to send to Kafka", zap.Error(err))
				continue
			}

			// 计算并更新下次时间
			schedule, err := parser.Parse(job.CronExpr)
			if err != nil {
				app.logger.Error("Invalid CronExpr", zap.Error(err))
				continue
			}
			nextTime := schedule.Next(now)
			app.data.DB.Model(&job).Update("next_time", nextTime.Unix())

			app.logger.Info("✅ Job rescheduled", zap.Uint("job_id", job.ID), zap.Time("next_run", nextTime))
		}
	}
}

func main() {
	app, cleanup, err := initApp()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	app.Run()
}
