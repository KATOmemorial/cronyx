package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/KATOmemorial/cronyx/internal/common"
	"github.com/KATOmemorial/cronyx/internal/model"
)

func main() {
	common.InitDB()

	consumer, err := sarama.NewConsumer([]string{"localhost:9092"}, nil)
	if err != nil {
		log.Fatalf("Failed to start Sarama consumer: %v", err)
	}
	defer consumer.Close()

	partitionConsumer, err := consumer.ConsumePartition("cronyx-jobs", 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatalf("Failed to start partition consumer: %v", err)
	}
	defer partitionConsumer.Close()

	fmt.Println("Worker started! Ready to execute jobs and report logs...")

	for msg := range partitionConsumer.Messages() {
		var event common.TaskEvent
		err := json.Unmarshal(msg.Value, &event)
		if err != nil {
			log.Printf("Error unmarshaling event: %v", err)
			continue
		}

		fmt.Printf("Executing Job: %s | Cmd: %s\n", event.TaskID, event.Command)

		parts := strings.Split(event.TaskID, "-")
		jobID, _ := strconv.Atoi(parts[0])

		jobLog := model.JobLog{
			JobID:     uint(jobID),
			Command:   event.Command,
			PlanTime:  event.Timestamp,
			RealTime:  time.Now().Unix(),
			StartTime: time.Now().UnixMilli(),
		}

		cmd := exec.Command("/bin/sh", "-c", event.Command)
		output, err := cmd.CombinedOutput()

		jobLog.EndTime = time.Now().UnixMilli()
		jobLog.Output = string(output)

		if err != nil {
			jobLog.Status = 0
			jobLog.Error = err.Error()
			fmt.Printf("Failed: %s\n", err)
		} else {
			jobLog.Status = 1
			fmt.Printf("Success: %s", string(output))
		}

		go func(logData model.JobLog) {
			if err := common.DB.Create(&logData).Error; err != nil {
				log.Printf("Failed to save log: %v", err)
			}
		}(jobLog)
	}
}
