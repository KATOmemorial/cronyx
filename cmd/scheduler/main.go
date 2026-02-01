package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
	"github.com/KATOmemorial/cronyx/internal/common"
)

func main() {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
	if err != nil {
		log.Fatalf("Failed to start Sarama producer: %v", err)
	}
	defer producer.Close()

	fmt.Println("sCHEDULER STARTED! Sending tasks every 5 seconds...")

	for {
		event := common.TaskEvent{
			TaskID:    "task-101",
			Command:   "echo 'HELLO Cronyx'",
			Timestamp: time.Now().Unix(),
		}

		bytes, _ := json.Marshal((event))

		msg := &sarama.ProducerMessage{
			Topic: "cronyx-jobs",
			Value: sarama.ByteEncoder(bytes),
		}

		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			log.Printf("failed to send message: %v", err)
		} else {
			fmt.Printf("[Scheduler] Sent task %s to Partition %d to Parition %d, Offset %d\n", event.TaskID, partition, offset)
		}

		time.Sleep(5 * time.Second)
	}
}
