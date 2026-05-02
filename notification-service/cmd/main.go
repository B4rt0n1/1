package main

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentCompletedEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type deduper struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func newDeduper() *deduper {
	return &deduper{seen: make(map[string]struct{})}
}

func (d *deduper) add(id string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.seen[id]; ok {
		return false
	}
	d.seen[id] = struct{}{}
	return true
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}

func main() {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@rabbitmq:5672/"
	}
	queueName := os.Getenv("RABBITMQ_QUEUE")
	if queueName == "" {
		queueName = "payment.completed"
	}

	conn, err := amqp.Dial(rabbitURL)
	failOnError(err, "failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "failed to open channel")
	defer ch.Close()

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "failed to declare queue")

	msgs, err := ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "failed to register consumer")

	dedup := newDeduper()
	log.Printf("Notification service started, waiting for messages on %s", queueName)

	forever := make(chan struct{})
	go func() {
		for d := range msgs {
			var event PaymentCompletedEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("Invalid event payload: %v", err)
				_ = d.Nack(false, false)
				continue
			}

			if event.EventID == "" {
				log.Printf("Missing event ID, skipping message")
				_ = d.Nack(false, false)
				continue
			}

			if !dedup.add(event.EventID) {
				log.Printf("[Notification] Duplicate event skipped for Order #%s", event.OrderID)
				_ = d.Ack(false)
				continue
			}

			log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%0.2f", event.CustomerEmail, event.OrderID, float64(event.Amount)/100)
			if err := d.Ack(false); err != nil {
				log.Printf("Failed to ack message: %v", err)
			}
		}
	}()

	<-forever
}
