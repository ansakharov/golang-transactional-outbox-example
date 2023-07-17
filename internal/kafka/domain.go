package kafka

import (
	"fmt"
	"log"

	"github.com/Shopify/sarama"
)

const Topic = "quickstart-events"

func NewProducer(port string) sarama.SyncProducer {
	// Конфигурация Kafka Producer
	cfg := sarama.NewConfig()
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Successes = true

	// Создаем Kafka Producer
	producer, err := sarama.NewSyncProducer([]string{fmt.Sprintf("localhost:%s", port)}, cfg)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	return producer
}
