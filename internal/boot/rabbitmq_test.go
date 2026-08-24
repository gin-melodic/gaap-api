package boot

import (
	"context"
	"sync"
	"testing"
	"time"

	"gaap-api/internal/mq"
)

type reconnectingMQ struct {
	mu           sync.Mutex
	connected    bool
	connectCalls int
}

func (m *reconnectingMQ) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func (m *reconnectingMQ) Connect(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	m.connectCalls++
	return nil
}

func (m *reconnectingMQ) Close() error {
	return nil
}

func (m *reconnectingMQ) Publish(context.Context, string, *mq.Message) error {
	return nil
}

func (m *reconnectingMQ) Consume(context.Context, string, func(context.Context, *mq.Message) error) error {
	return nil
}

func (m *reconnectingMQ) calls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connectCalls
}

func TestMaintainRabbitMQConnectionReconnectsAndRestartsConsumers(t *testing.T) {
	client := &reconnectingMQ{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	restarted := make(chan struct{}, 1)
	go maintainRabbitMQConnection(ctx, client, time.Millisecond, func() {
		restarted <- struct{}{}
	})

	select {
	case <-restarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for RabbitMQ reconnect")
	}

	if !client.IsConnected() {
		t.Fatal("RabbitMQ client was not reconnected")
	}
	if got := client.calls(); got != 1 {
		t.Fatalf("Connect() calls = %d, want 1", got)
	}
}

func TestMaintainRabbitMQConnectionDoesNotReconnectHealthyClient(t *testing.T) {
	client := &reconnectingMQ{connected: true}
	ctx, cancel := context.WithCancel(context.Background())

	go maintainRabbitMQConnection(ctx, client, time.Millisecond, func() {
		t.Error("healthy client should not restart consumers")
	})
	time.Sleep(10 * time.Millisecond)
	cancel()

	if got := client.calls(); got != 0 {
		t.Fatalf("Connect() calls = %d, want 0", got)
	}
}
