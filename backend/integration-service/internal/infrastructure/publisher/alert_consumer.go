package publisher

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/raul/monitor/backend/integration-service/internal/service/webhook"
	"github.com/raul/monitor/backend/integration-service/internal/service/webhook/dto"
)

// AlertConsumer потребитель событий алертов от Alert Service.
type AlertConsumer struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	queue        amqp.Queue
	deliverySvc  *webhook.WebhookDeliveryService
	exchangeName string
	routingKey   string
	log          *slog.Logger
}

// AlertConsumerConfig конфигурация consumer.
type AlertConsumerConfig struct {
	ExchangeName string
	RoutingKey   string
	QueueName    string
}

// NewAlertConsumer создаёт нового AlertConsumer.
func NewAlertConsumer(
	conn *amqp.Connection,
	deliverySvc *webhook.WebhookDeliveryService,
	config *AlertConsumerConfig,
) *AlertConsumer {
	if config == nil {
		config = &AlertConsumerConfig{
			ExchangeName: "monitor",
			RoutingKey:   "alert.triggered",
			QueueName:    "integration_service_webhooks",
		}
	}

	return &AlertConsumer{
		conn:         conn,
		deliverySvc:  deliverySvc,
		exchangeName: config.ExchangeName,
		routingKey:   config.RoutingKey,
		log:          slog.Default(),
	}
}

// Start запускает consumer.
func (c *AlertConsumer) Start(ctx context.Context) error {
	c.log.InfoContext(ctx, "starting alert consumer",
		"exchange", c.exchangeName,
		"routing_key", c.routingKey,
	)

	var err error
	c.channel, err = c.conn.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to create channel")
	}

	// Объявляем exchange
	err = c.channel.ExchangeDeclare(
		c.exchangeName,
		"topic", // Topic exchange для routing key patterns
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // noWait
		nil,     // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed to declare exchange")
	}

	// Объявляем очередь
	queueName := "integration_service_webhooks" // default queue name
	c.queue, err = c.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // auto-delete
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return errors.Wrap(err, "failed to declare queue")
	}

	// Привязываем очередь к exchange с routing key
	err = c.channel.QueueBind(
		c.queue.Name,
		c.routingKey,
		c.exchangeName,
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return errors.Wrap(err, "failed to bind queue")
	}

	// Устанавливаем QoS
	err = c.channel.Qos(
		10,    // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return errors.Wrap(err, "failed to set QoS")
	}

	// Начинаем потреблять сообщения
	msgs, err := c.channel.Consume(
		c.queue.Name,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return errors.Wrap(err, "failed to start consuming")
	}

	c.log.InfoContext(ctx, "alert consumer started",
		"queue", c.queue.Name,
	)

	// Обрабатываем сообщения в горутине
	go c.consumeMessages(ctx, msgs)

	return nil
}

// Stop останавливает consumer.
func (c *AlertConsumer) Stop(ctx context.Context) error {
	c.log.InfoContext(ctx, "stopping alert consumer")

	if err := c.channel.Cancel("", false); err != nil {
		return errors.Wrap(err, "failed to cancel consumer")
	}

	// Ждём завершения обработки текущих сообщений (graceful shutdown)
	c.log.InfoContext(ctx, "waiting for in-flight messages to complete")

	// Создаём timeout для graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Ожидаем завершения горутины consumeMessages
	// Она автоматически завершится при cancel контекста
	<-ctx.Done()

	// Дополнительное время для завершения текущих сообщений
	select {
	case <-shutdownCtx.Done():
		if shutdownCtx.Err() == context.DeadlineExceeded {
			c.log.WarnContext(ctx, "graceful shutdown timeout exceeded, forcing close")
		}
	case <-time.After(100 * time.Millisecond):
		// Даём время на завершение
	}

	return nil
}

// consumeMessages обрабатывает входящие сообщения.
func (c *AlertConsumer) consumeMessages(ctx context.Context, msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			c.log.InfoContext(ctx, "alert consumer stopped")
			return

		case msg, ok := <-msgs:
			if !ok {
				c.log.WarnContext(ctx, "alert consumer channel closed")
				return
			}

			c.processMessage(ctx, msg)
		}
	}
}

// processMessage обрабатывает одно сообщение.
func (c *AlertConsumer) processMessage(ctx context.Context, msg amqp.Delivery) {
	c.log.DebugContext(ctx, "received alert event",
		"content_type", msg.ContentType,
		"body_length", len(msg.Body),
	)

	// Парсим событие
	event, err := dto.ParseAlertEventFromJSON(msg.Body)
	if err != nil {
		c.log.ErrorContext(ctx, "failed to parse alert event",
			"error", err,
			"body", string(msg.Body),
		)
		if nackErr := msg.Nack(false, false); nackErr != nil {
			c.log.ErrorContext(ctx, "failed to nack malformed alert event", "error", nackErr)
		}
		return
	}

	// Обрабатываем событие
	if err := c.deliverySvc.ProcessAlertEvent(ctx, event); err != nil {
		c.log.ErrorContext(ctx, "failed to process alert event",
			"alert_id", event.AlertID,
			"error", err,
		)
		if nackErr := msg.Nack(false, false); nackErr != nil {
			c.log.ErrorContext(ctx, "failed to nack alert event", "error", nackErr)
		}
		return
	}

	// Успешно обработано - acknowledge
	if ackErr := msg.Ack(false); ackErr != nil {
		c.log.ErrorContext(ctx, "failed to ack processed alert event", "error", ackErr)
	}

	c.log.InfoContext(ctx, "alert event processed successfully",
		"alert_id", event.AlertID,
		"monitor_id", event.MonitorID,
	)
}
