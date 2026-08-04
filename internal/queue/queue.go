// Package queue provides RabbitMQ-backed delayed notification delivery with exponential retry.
package queue

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/venexene/gong/internal/repository"
)

// Publisher enqueues a notification for delayed delivery.
type Publisher interface {
	Publish(n repository.Notification) error
}

// Notifier delivers a notification through a specific channel.
type Notifier interface {
	Send(n repository.Notification) error
}

// LogNotifier is a Notifier that logs the notification to stdout.
type LogNotifier struct{}

// Send logs the notification details and always returns nil.
func (l LogNotifier) Send(n repository.Notification) error {
	log.Printf("[SENT] target=%s message=%s id=%s", n.Target, n.Message, n.ID)
	return nil
}

// RabbitMQ manages AMQP queues for delayed and retried notifications.
//
// Queue topology:
//
//	notifications        – main processing queue consumed by the worker.
//	notifications_delay  – TTL-based delay queue; expired messages are
//	                        dead-lettered to the main queue.
//	notifications_retry   – exponential backoff retry queue; expired messages
//	                        are dead-lettered back to the main queue.
type RabbitMQ struct {
	Conn       *amqp.Connection
	Channel    *amqp.Channel
	MainQueue  amqp.Queue
	DelayQueue amqp.Queue
	RetryQueue amqp.Queue
}

const (
	// BaseRetryDelay is the initial delay before the first retry.
	BaseRetryDelay = 5 * time.Second
	// MaxRetryDelay is the upper cap for exponential backoff.
	MaxRetryDelay = 10 * time.Minute
	// MaxRetries is the number of delivery attempts before marking as failed.
	MaxRetries = 10
)

// New connects to RabbitMQ at the given URL and declares all required queues.
func New(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	mainQueue, err := ch.QueueDeclare(
		"notifications",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	delayQueue, err := ch.QueueDeclare(
		"notifications_delay",
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": mainQueue.Name,
		},
	)
	if err != nil {
		return nil, err
	}

	retryQueue, err := ch.QueueDeclare(
		"notifications_retry",
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": mainQueue.Name,
		},
	)
	if err != nil {
		return nil, err
	}

	log.Println("RabbitMQ connected")

	return &RabbitMQ{
		Conn:       conn,
		Channel:    ch,
		MainQueue:  mainQueue,
		DelayQueue: delayQueue,
		RetryQueue: retryQueue,
	}, nil
}

// Publish enqueues a notification for delayed delivery.
func (r *RabbitMQ) Publish(n repository.Notification) error {
	body, err := json.Marshal(n)
	if err != nil {
		return err
	}

	delay := time.Until(n.SendAt)
	if delay < 0 {
		delay = 0
	}

	return r.Channel.Publish(
		"",
		r.DelayQueue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Expiration:  strconv.FormatInt(delay.Milliseconds(), 10),
		},
	)
}

// PublishRetry enqueues a failed notification for retry with exponential backoff.
func (r *RabbitMQ) PublishRetry(n repository.Notification, retry int) error {
	body, err := json.Marshal(n)
	if err != nil {
		return err
	}

	delay := calcRetryDelay(retry)

	log.Printf("Retry %d for %s in %s", retry, n.ID, delay)

	return r.Channel.Publish(
		"",
		r.RetryQueue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Expiration:  strconv.FormatInt(delay.Milliseconds(), 10),
		},
	)
}

// Consume starts a blocking message consumer on the main queue.
func (r *RabbitMQ) Consume(ctx context.Context, db *repository.Postgres, notifier Notifier) {
	msgs, err := r.Channel.Consume(
		r.MainQueue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to consume: %v", err)
	}

	log.Println("Worker is consuming messages...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer stopping...")
			return

		case msg, ok := <-msgs:
			if !ok {
				log.Println("Message channel closed")
				return
			}

			var n repository.Notification
			if err := json.Unmarshal(msg.Body, &n); err != nil {
				log.Println("Invalid message body")
				nack(msg, false, false)
				continue
			}

			current, err := db.GetByID(ctx, n.ID)
			if err != nil {
				log.Printf("Notification %s not found in DB", n.ID)
				ack(msg, false)
				continue
			}

			if current.Status == "canceled" || current.Status == "sent" {
				ack(msg, false)
				continue
			}

			if current.Retry >= MaxRetries {
				log.Printf("Notification %s failed permanently", n.ID)
				if err := db.MarkFailed(ctx, n.ID); err != nil {
					log.Printf("DB MarkFailed error: %v", err)
				}
				ack(msg, false)
				continue
			}

			if err := db.MarkProcessing(ctx, n.ID); err != nil {
				log.Printf("Cannot mark processing: %v", err)
				nack(msg, false, true)
				continue
			}

			if err := notifier.Send(n); err != nil {
				log.Printf("Send failed, retrying: %v", err)
				if err := db.IncrementRetry(ctx, n.ID); err != nil {
					log.Printf("DB IncrementRetry error: %v", err)
				}
				updated, err := db.GetByID(ctx, n.ID)
				if err != nil {
					log.Printf("DB GetByID error: %v", err)
					nack(msg, false, true)
					continue
				}
				if err := r.PublishRetry(*updated, updated.Retry); err != nil {
					log.Printf("PublishRetry error (will requeue): %v", err)
					nack(msg, false, true)
					continue
				}
				ack(msg, false)
				continue
			}

			if err := db.MarkSent(ctx, n.ID); err != nil {
				log.Printf("DB MarkSent error: %v", err)
			}

			ack(msg, false)
		}
	}
}

func calcRetryDelay(retry int) time.Duration {
	delay := float64(BaseRetryDelay) * math.Pow(2, float64(retry-1))

	if delay > float64(MaxRetryDelay) {
		delay = float64(MaxRetryDelay)
	}

	return time.Duration(delay)
}

func nack(msg amqp.Delivery, multiple, requeue bool) {
	if err := msg.Nack(multiple, requeue); err != nil {
		log.Printf("msg.Nack error: %v", err)
	}
}

func ack(msg amqp.Delivery, multiple bool) {
	if err := msg.Ack(multiple); err != nil {
		log.Printf("msg.Ack error: %v", err)
	}
}
