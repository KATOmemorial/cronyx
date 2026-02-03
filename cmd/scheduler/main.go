package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/model"
	"github.com/robfig/cron/v3"
)

func main() {
	common.InitDB()

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
	if err != nil {
		log.Fatalf("Failed to start Kafka producer: %v", err)
	}
	defer producer.Close()

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	fmt.Println("Smart Scheduler started! Scanning DB every second...")

	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		var jobs []model.JobInfo
		now := time.Now()

		if err := common.DB.Where("status = ? AND next_time <= ?", 1, now.Unix()).Find(&jobs).Error; err != nil {
			log.Printf("Failed to fetch jobs: %v", err)
			continue
		}

		if len(jobs) == 0 {
			continue
		}

		fmt.Printf("Found %d tasks to trigger...\n", len(jobs))

		for _, job := range jobs {
			event := common.TaskEvent{
				TaskID:    fmt.Sprintf("%d-%d", job.ID, now.Unix()),
				Command:   job.Command,
				Timestamp: now.Unix(),
			}
			bytes, _ := json.Marshal(event)

			msg := &sarama.ProducerMessage{
				Topic: "cronyx-jobs",
				Value: sarama.ByteEncoder(bytes),
			}

			_, _, err := producer.SendMessage(msg)
			if err != nil {
				log.Printf("Failed to send task %d: %v", job.ID, err)
				continue
			}
			fmt.Printf("Triggered job: %s (ID: %d)\n", job.Name, job.ID)

			schedule, err := parser.Parse(job.CronExpr)
			if err != nil {
				log.Printf("Invalid CronExpr for job %d: %v", job.ID, err)
				continue
			}

			nextTime := schedule.Next(now)

			common.DB.Model(&job).Update("next_time", nextTime.Unix())
			fmt.Printf("Refreshed next execution time: %s\n", nextTime.Format("15:04:05"))
		}
	}
}
