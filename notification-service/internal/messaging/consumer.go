package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"notification-service/internal/domain"
	"notification-service/internal/idempotency"
	"notification-service/internal/notifier"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	queueName   = "payment.completed"
	dlxName     = "payment.dlx"
	dlqName     = "payment.completed.dlq"
	dlqRouteKey = "dead"
)

type Consumer struct {
	conn          *amqp.Connection
	channel       *amqp.Channel
	store         idempotency.Store
	retries       map[string]int
	sender        notifier.EmailSender
	maxRetries    int
	baseDelayMs   int
}

func NewConsumer(url string, store idempotency.Store, sender notifier.EmailSender, maxRetries, baseDelayMs int) (*Consumer, error) {
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

	return &Consumer{
		conn:        conn,
		channel:     ch,
		store:       store,
		retries:     make(map[string]int),
		sender:      sender,
		maxRetries:  maxRetries,
		baseDelayMs: baseDelayMs,
	}, nil
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
	ctx := context.Background()

	var event domain.PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[Notification] failed to unmarshal message: %v — sending to DLQ", err)
		msg.Nack(false, false)
		return
	}

	seen, err := c.store.Seen(ctx, event.EventID)
	if err != nil {
		log.Printf("[Notification] idempotency check failed for event %s: %v — requeueing", event.EventID, err)
		msg.Nack(false, true)
		return
	}
	if seen {
		log.Printf("[Notification] duplicate event %s — skipping", event.EventID)
		msg.Ack(false)
		return
	}

	if err := c.sendNotification(ctx, event); err != nil {
		attempt := c.retries[event.EventID] + 1
		c.retries[event.EventID] = attempt

		if attempt >= c.maxRetries {
			log.Printf("[DLQ] Message for Order #%s moved to dead letter queue after %d retries", event.OrderID, c.maxRetries)
			delete(c.retries, event.EventID)
			msg.Nack(false, false)
		} else {
			// Exponential backoff: baseDelay * 2^(attempt-1)
			delay := time.Duration(c.baseDelayMs) * time.Millisecond * (1 << (attempt - 1))
			log.Printf("[Notification] error processing event %s (attempt %d/%d): %v — retrying in %s",
				event.EventID, attempt, c.maxRetries, err, delay)
			time.Sleep(delay)
			msg.Nack(false, true)
		}
		return
	}

	if err := c.store.Mark(ctx, event.EventID); err != nil {
		log.Printf("[Notification] failed to mark event %s as processed: %v", event.EventID, err)
	}
	delete(c.retries, event.EventID)
	msg.Ack(false)
}

func (c *Consumer) sendNotification(ctx context.Context, event domain.PaymentEvent) error {
	amountDollars := float64(event.Amount) / 100.0
	log.Printf("[Notification] Sending email to %s for Order #%s. Amount: $%.2f", event.CustomerEmail, event.OrderID, amountDollars)

	if c.sender != nil {
		subject := fmt.Sprintf("Payment %s for Order #%s", event.Status, event.OrderID)
		body := fmt.Sprintf("Your payment of $%.2f for order #%s has been %s.", amountDollars, event.OrderID, event.Status)
		if err := c.sender.Send(ctx, event.CustomerEmail, subject, body); err != nil {
			return fmt.Errorf("send email: %w", err)
		}
	}

	return nil
}

func (c *Consumer) Close() {
	c.channel.Close()
	c.conn.Close()
}
