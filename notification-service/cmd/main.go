package main

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	defaultQueueName       = "payment.completed"
	deadLetterExchangeName = "payment.completed.dlx"
	deadLetterQueueName    = "payment.completed.dlq"
	maxRetryAttempts       = 3
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

func getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}

	switch v := headers["x-retry-count"].(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func publishWithRetryHeaders(ch *amqp.Channel, queueName string, headers amqp.Table, body []byte) error {
	if headers == nil {
		headers = amqp.Table{}
	}

	return ch.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
			Body:         body,
		},
	)
}

func processNotification(event *PaymentCompletedEvent) error {
	if event.EventID == "" {
		return errors.New("missing event id")
	}
	if event.OrderID == "" {
		return errors.New("missing order id")
	}
	if event.CustomerEmail == "" {
		return errors.New("missing customer email")
	}
	return nil
}

func main() {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@rabbitmq:5672/"
	}
	queueName := os.Getenv("RABBITMQ_QUEUE")
	if queueName == "" {
		queueName = defaultQueueName
	}

	conn, err := amqp.Dial(rabbitURL)
	failOnError(err, "failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "failed to open channel")
	defer ch.Close()

	failOnError(ch.ExchangeDeclare(deadLetterExchangeName, "direct", true, false, false, false, nil), "failed to declare dead letter exchange")

	_, err = ch.QueueDeclare(
		deadLetterQueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "failed to declare dead letter queue")

	failOnError(ch.QueueBind(deadLetterQueueName, deadLetterQueueName, deadLetterExchangeName, false, nil), "failed to bind dead letter queue")

	queueArgs := amqp.Table{
		"x-dead-letter-exchange":    deadLetterExchangeName,
		"x-dead-letter-routing-key": deadLetterQueueName,
	}

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		queueArgs,
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

			retryCount := getRetryCount(d.Headers)
			if err := processNotification(&event); err != nil {
				if retryCount >= maxRetryAttempts {
					log.Printf("Exceeded retry attempts for Order #%s, moving to DLQ: %v", event.OrderID, err)
					_ = d.Nack(false, false)
					continue
				}

				if d.Headers == nil {
					d.Headers = amqp.Table{}
				}
				d.Headers["x-retry-count"] = retryCount + 1
				if err := publishWithRetryHeaders(ch, queueName, d.Headers, d.Body); err != nil {
					log.Printf("Failed to republish message for retry: %v", err)
					_ = d.Nack(false, false)
					continue
				}

				log.Printf("Retrying message for Order #%s (attempt %d)", event.OrderID, retryCount+1)
				_ = d.Ack(false)
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
