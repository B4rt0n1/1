package broker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type EventPublisher interface {
	PublishPaymentCompleted(ctx context.Context, event *PaymentEvent) error
	Close() error
}

type RabbitMQPublisher struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	confirms  <-chan amqp.Confirmation
	queueName string
}

func NewRabbitMQPublisher(rabbitURL, queueName string) (*RabbitMQPublisher, error) {
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@rabbitmq:5672/"
	}
	if queueName == "" {
		queueName = "payment.completed"
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}

	if err := ch.Confirm(false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	return &RabbitMQPublisher{
		conn:      conn,
		channel:   ch,
		confirms:  confirms,
		queueName: queueName,
	}, nil
}

func (p *RabbitMQPublisher) PublishPaymentCompleted(ctx context.Context, event *PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	if err := p.channel.Publish(
		"",
		p.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			MessageId:    event.EventID,
			Body:         body,
		},
	); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	select {
	case confirm := <-p.confirms:
		if !confirm.Ack {
			return errors.New("broker rejected published message")
		}
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("publisher confirm timeout")
	}
}

func (p *RabbitMQPublisher) Close() error {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
