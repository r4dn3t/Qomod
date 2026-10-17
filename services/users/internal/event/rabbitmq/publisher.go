package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"qomod/services/users/internal/domain"
)

const (
	DefaultExchangeName = "users.events"
	ExchangeType        = "topic"
)

// Config holds RabbitMQ connection parameters.
type Config struct {
	URL          string
	ExchangeName string
}

// Publisher publishes domain events to RabbitMQ topic exchange.
// It is thread-safe and implements automatic reconnection.
type Publisher struct {
	cfg        Config
	logger     *slog.Logger
	mu         sync.RWMutex
	conn       *amqp.Connection
	ch         *amqp.Channel
	isClosed   bool
	notifyClose chan *amqp.Error
}

// NewPublisher creates and connects a new RabbitMQ event publisher.
func NewPublisher(ctx context.Context, cfg Config, logger *slog.Logger) (*Publisher, error) {
	if cfg.ExchangeName == "" {
		cfg.ExchangeName = DefaultExchangeName
	}

	p := &Publisher{
		cfg:    cfg,
		logger: logger,
	}

	if err := p.connect(); err != nil {
		return nil, fmt.Errorf("initial rabbitmq connection failed: %w", err)
	}

	go p.handleReconnect()

	return p, nil
}

// connect initializes connection, channel, and declares the topic exchange.
func (p *Publisher) connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	conn, err := amqp.Dial(p.cfg.URL)
	if err != nil {
		return fmt.Errorf("dial amqp failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel failed: %w", err)
	}

	// Declare topic exchange
	err = ch.ExchangeDeclare(
		p.cfg.ExchangeName, // exchange name
		ExchangeType,       // exchange type
		true,               // durable
		false,              // auto-deleted
		false,              // internal
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("exchange declare failed: %w", err)
	}

	p.conn = conn
	p.ch = ch
	p.notifyClose = make(chan *amqp.Error, 1)
	p.ch.NotifyClose(p.notifyClose)

	p.logger.Info("connected to rabbitmq exchange", "exchange", p.cfg.ExchangeName)
	return nil
}

// handleReconnect watches for connection closure and attempts reconnection in the background.
func (p *Publisher) handleReconnect() {
	for {
		p.mu.RLock()
		if p.isClosed {
			p.mu.RUnlock()
			return
		}
		notify := p.notifyClose
		p.mu.RUnlock()

		err, ok := <-notify
		if !ok {
			// Channel was cleanly closed or connection closing
			return
		}

		p.logger.Warn("rabbitmq connection lost, attempting reconnect...", "error", err)

		backoff := 1 * time.Second
		for {
			p.mu.RLock()
			closed := p.isClosed
			p.mu.RUnlock()
			if closed {
				return
			}

			time.Sleep(backoff)
			if reconnErr := p.connect(); reconnErr == nil {
				p.logger.Info("rabbitmq successfully reconnected")
				break
			}

			if backoff < 30*time.Second {
				backoff *= 2
			}
			p.logger.Warn("rabbitmq reconnect attempt failed, retrying...", "backoff", backoff)
		}
	}
}

// Publish sends a domain event enveloped in JSON to the specified routing key.
func (p *Publisher) Publish(ctx context.Context, routingKey string, event domain.EventEnvelope) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.isClosed {
		return fmt.Errorf("publisher is closed")
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event JSON: %w", err)
	}

	msg := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent, // Ensure messages survive RabbitMQ restarts
		MessageId:    event.EventID,
		Timestamp:    event.Timestamp,
		Type:         event.EventType,
		Body:         body,
	}

	err = p.ch.PublishWithContext(
		ctx,
		p.cfg.ExchangeName, // exchange
		routingKey,         // routing key
		false,              // mandatory
		false,              // immediate
		msg,
	)
	if err != nil {
		return fmt.Errorf("rabbitmq publish failed: %w", err)
	}

	p.logger.Debug("published event", "type", event.EventType, "routing_key", routingKey, "id", event.EventID)
	return nil
}

// Close gracefully closes the channel and connection.
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.isClosed = true

	var chErr, connErr error
	if p.ch != nil {
		chErr = p.ch.Close()
	}
	if p.conn != nil {
		connErr = p.conn.Close()
	}

	if chErr != nil {
		return chErr
	}
	return connErr
}
