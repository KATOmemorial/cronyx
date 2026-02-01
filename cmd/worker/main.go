package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	"github.com/IBM/sarama"
	"github.com/KATOmemorial/cronyx/internal/common"
)

func main() {
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

	fmt.Println("Worker started! Waiting for tasks...")

	for msg := range partitionConsumer.Messages() {
		var event common.TaskEvent
		err := json.Unmarshal(msg.Value, &event)
		if err != nil {
			log.Printf("Error unmarshaling event: %v", err)
			continue
		}

		fmt.Printf("[Worker] Received task: %s | Command: %s\n", event.TaskID, event.Command)

		cmd := exec.Command("/bin/sh", "-c", event.Command)
		output, err := cmd.CombinedOutput()

		if err != nil {
			fmt.Printf("Execution Failed: %d\n", err)
		} else {
			fmt.Printf("Output: %s", string(output))
		}
	}
}
