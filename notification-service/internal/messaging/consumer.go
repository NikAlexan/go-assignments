package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"notification-service/internal/domain"
	"notification-service/internal/idempotency"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	queueName   = "payment.completed"
	dlxName     = "payment.dlx"
	dlqName     = "payment.completed.dlq"
	dlqRouteKey = "dead"
	maxRetries  = 3
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	store   *idempotency.Store
	retries map[string]int
}

func NewConsumer(url string, store *idempotency.Store) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	if err := declareTopology(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	// prefetch 1 so we process one message at a time
	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("set qos: %w", err)
	}

	return &Consumer{conn: conn, channel: ch, store: store, retries: make(map[string]int)}, nil
}

func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(dlxName, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx: %w", err)
	}

	if _, err := ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlq: %w", err)
	}
	if err := ch.QueueBind(dlqName, dlqRouteKey, dlxName, false, nil); err != nil {
		return fmt.Errorf("bind dlq: %w", err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange":    dlxName,
		"x-dead-letter-routing-key": dlqRouteKey,
	}
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	return nil
}

func (c *Consumer) Start(done <-chan struct{}) error {
	msgs, err := c.channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	log.Println("notification-service: waiting for messages...")

	for {
		select {
		case <-done:
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			c.handle(msg)
		}
	}
}

func (c *Consumer) handle(msg amqp.Delivery) {
	var event domain.PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[Notification] failed to unmarshal message: %v — sending to DLQ", err)
		msg.Nack(false, false)
		return
	}

	if c.store.Seen(event.EventID) {
		log.Printf("[Notification] duplicate event %s — skipping", event.EventID)
		msg.Ack(false)
		return
	}

	if err := c.sendNotification(event); err != nil {
		c.retries[event.EventID]++
		if c.retries[event.EventID] >= maxRetries {
			log.Printf("[DLQ] Message for Order #%s moved to dead letter queue after %d retries", event.OrderID, maxRetries)
			delete(c.retries, event.EventID)
			msg.Nack(false, false)
		} else {
			log.Printf("[Notification] error processing event %s (attempt %d): %v — requeueing", event.EventID, c.retries[event.EventID], err)
			msg.Nack(false, true)
		}
		return
	}

	c.store.Mark(event.EventID)
	delete(c.retries, event.EventID)
	msg.Ack(false)
}

func (c *Consumer) sendNotification(event domain.PaymentEvent) error {
	amountDollars := float64(event.Amount) / 100.0
	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%.2f", event.CustomerEmail, event.OrderID, amountDollars)
	return nil
}

func (c *Consumer) Close() {
	c.channel.Close()
	c.conn.Close()
}
