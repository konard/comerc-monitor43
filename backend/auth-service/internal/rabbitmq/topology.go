package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	otelcodes "go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("github.com/raul/monitor/backend/auth-service/internal/rabbitmq")

// channelOpener открывает новый AMQP-канал.
type channelOpener interface {
	Channel() (*amqp.Channel, error)
}

// amqpChannel описывает AMQP-операции, используемые при декларировании топологии.
type amqpChannel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	Close() error
}

// Config содержит параметры AMQP-топологии.
type Config struct {
	Exchange   string
	Queue      string
	RoutingKey string
}

// Topology декларирует AMQP-топологию через dialer.
type Topology struct {
	cfg    Config
	dialer channelOpener
}

// New создаёт Topology; тяжёлая инициализация откладывается до Start.
func New(cfg Config, dialer channelOpener) *Topology {
	return &Topology{cfg: cfg, dialer: dialer}
}

// Start декларирует exchange, очередь и binding.
// Вызывать при старте приложения после подключения к брокеру и до запуска publisher.
func (t *Topology) Start(ctx context.Context) (err error) {
	ctx, span := tracer.Start(ctx, "rabbitmq.Topology.Start")
	defer span.End()

	ch, err := t.dialer.Channel()
	if err != nil {
		err = fmt.Errorf("open channel: %w", err)
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
		return err
	}
	defer func() {
		if closeErr := ch.Close(); closeErr != nil {
			closeErr = fmt.Errorf("close channel: %w", closeErr)
			span.RecordError(closeErr)
			if err == nil {
				err = closeErr
				span.SetStatus(otelcodes.Error, closeErr.Error())
			}
		}
	}()

	if err = t.declare(ctx, ch); err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
	}
	return err
}

// declare выполняет последовательность AMQP-деклараций на уже открытом канале.
func (t *Topology) declare(ctx context.Context, ch amqpChannel) error {
	_, span := tracer.Start(ctx, "rabbitmq.Topology.declare")
	defer span.End()

	// 1. Exchange (topic, durable)
	if err := ch.ExchangeDeclare(t.cfg.Exchange, "topic", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange %q: %w", t.cfg.Exchange, err)
	}

	// 2. Очередь событий (durable)
	if _, err := ch.QueueDeclare(t.cfg.Queue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue %q: %w", t.cfg.Queue, err)
	}

	// 3. Binding: очередь → exchange
	if err := ch.QueueBind(t.cfg.Queue, t.cfg.RoutingKey, t.cfg.Exchange, false, nil); err != nil {
		return fmt.Errorf("bind queue %q to exchange %q: %w", t.cfg.Queue, t.cfg.Exchange, err)
	}

	return nil
}
