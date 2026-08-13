package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Message represents a message to be published or consumed
type Message struct {
	Type    int             `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Client interface for RabbitMQ to allow mocking
type Client interface {
	IsConnected() bool
	Connect(ctx context.Context) error
	Close() error
	Publish(ctx context.Context, queue string, msg *Message) error
	Consume(ctx context.Context, queue string, handler func(ctx context.Context, msg *Message) error) error
}

// RabbitMQ manages RabbitMQ connections and channels
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	mu      sync.RWMutex
	url     string
}

var (
	clientInstance Client
	once           sync.Once
)

// Queue names
const (
	QueueTasks     = "gaap.tasks"
	QueueDashboard = "gaap.dashboard"
)

// GetRabbitMQ returns singleton RabbitMQ client
func GetRabbitMQ() Client {
	once.Do(func() {
		if clientInstance == nil {
			clientInstance = &RabbitMQ{}
		}
	})
	// fmt.Printf("GetRabbitMQ called, returning type: %T\n", clientInstance)
	return clientInstance
}

// SetClient sets the global RabbitMQ client (for testing)
func SetClient(c Client) {
	fmt.Printf("SetClient called with type: %T\n", c)
	clientInstance = c
	// Reset once to allow re-initialization if needed, though usually for tests we just set it once
}

// IsConnected returns true if RabbitMQ channel is initialized and ready
func (r *RabbitMQ) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.conn != nil && !r.conn.IsClosed() && r.channel != nil && !r.channel.IsClosed()
}

// Connect establishes connection to RabbitMQ with retry logic
func (r *RabbitMQ) Connect(ctx context.Context) error {
	// Build connection URL from environment
	host := os.Getenv("RABBITMQ_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("RABBITMQ_PORT")
	if port == "" {
		port = "5672"
	}
	user := os.Getenv("RABBITMQ_USER")
	if user == "" {
		user = "guest"
	}
	pass := os.Getenv("RABBITMQ_PASSWORD")
	if pass == "" {
		pass = "guest"
	}

	r.url = (&url.URL{
		Scheme: "amqp",
		User:   url.UserPassword(user, pass),
		Host:   net.JoinHostPort(host, port),
		Path:   "/",
	}).String()

	// Retry connection with exponential backoff
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		err := r.tryConnect(ctx)
		if err == nil {
			g.Log().Info(ctx, "RabbitMQ connected successfully")
			return nil
		}

		if i < maxRetries-1 {
			waitTime := time.Duration(1<<uint(i)) * time.Second // 1s, 2s, 4s, 8s, 16s
			g.Log().Warningf(ctx, "RabbitMQ connection attempt %d failed, retrying in %v: %v", i+1, waitTime, err)
			timer := time.NewTimer(waitTime)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return ctx.Err()
			case <-timer.C:
			}
		} else {
			return fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", maxRetries, err)
		}
	}
	return nil
}

// tryConnect attempts a single connection to RabbitMQ
func (r *RabbitMQ) tryConnect(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var err error
	r.conn, err = amqp.Dial(r.url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	r.channel, err = r.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare queues
	for _, queueName := range []string{QueueTasks, QueueDashboard} {
		_, err = r.channel.QueueDeclare(
			queueName, // name
			true,      // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			nil,       // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
		}
	}

	return nil
}

// Close closes the connection
func (r *RabbitMQ) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// Publish sends a message to the specified queue
func (r *RabbitMQ) Publish(ctx context.Context, queue string, msg *Message) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.channel == nil {
		return fmt.Errorf("RabbitMQ channel not initialized")
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = r.channel.PublishWithContext(ctx,
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
		})
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	g.Log().Debugf(ctx, "Published message to queue %s: %d", queue, msg.Type)
	return nil
}

// Consume starts consuming messages from the specified queue
func (r *RabbitMQ) Consume(ctx context.Context, queue string, handler func(ctx context.Context, msg *Message) error) error {
	r.mu.RLock()
	channel := r.channel
	r.mu.RUnlock()

	if channel == nil {
		return fmt.Errorf("RabbitMQ channel not initialized")
	}

	msgs, err := channel.Consume(
		queue, // queue
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-msgs:
				if !ok {
					return
				}

				var msg Message
				if err := json.Unmarshal(d.Body, &msg); err != nil {
					g.Log().Errorf(ctx, "Failed to unmarshal message: %v", err)
					d.Nack(false, false)
					continue
				}

				if err := handler(ctx, &msg); err != nil {
					g.Log().Errorf(ctx, "Failed to process message: %v", err)
					d.Nack(false, true) // requeue on failure
				} else {
					d.Ack(false)
				}
			}
		}
	}()

	g.Log().Infof(ctx, "Started consuming from queue: %s", queue)
	return nil
}
