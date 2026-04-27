package publisher

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raul/monitor/backend/integration-service/internal/model"
	"github.com/raul/monitor/backend/integration-service/internal/repository/interfaces"
	"github.com/raul/monitor/backend/integration-service/internal/service/webhook"
)

func stopDeliveryService(tb testing.TB, svc *webhook.WebhookDeliveryService) {
	tb.Helper()

	tb.Cleanup(func() {
		if err := svc.Stop(context.Background()); err != nil {
			tb.Errorf("stop delivery service: %v", err)
		}
	})
}

func closeNetConn(tb testing.TB, conn net.Conn) {
	tb.Helper()

	if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		tb.Logf("net.Conn close: %v", err)
	}
}

// ---- минимальный фейковый AMQP-сервер для unit-тестов Start/Stop ----

// writeAMQPFrame записывает один AMQP-фрейм: тип + канал + размер + payload + 0xCE.
func writeAMQPFrame(w io.Writer, typ uint8, channel uint16, payload []byte) error {
	hdr := []byte{
		typ,
		byte(channel >> 8), byte(channel & 0xFF), //nolint:gosec // G115: channel≤65535
		byte(len(payload) >> 24), byte(len(payload) >> 16), //nolint:gosec // G115: AMQP max frame 128MB
		byte(len(payload) >> 8), byte(len(payload) & 0xFF), //nolint:gosec // G115
	}
	if _, err := w.Write(hdr); err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		return err
	}
	_, err := w.Write([]byte{0xCE}) // frame-end
	return err
}

// readAMQPFrame читает следующий фрейм и возвращает тип, канал и payload.
func readAMQPFrame(r io.Reader) (typ uint8, channel uint16, payload []byte, err error) {
	hdr := make([]byte, 7)
	if _, err = io.ReadFull(r, hdr); err != nil {
		return typ, channel, payload, err
	}
	typ = hdr[0]
	channel = binary.BigEndian.Uint16(hdr[1:3])
	size := binary.BigEndian.Uint32(hdr[3:7])
	payload = make([]byte, size)
	if _, err = io.ReadFull(r, payload); err != nil {
		return typ, channel, payload, err
	}
	end := make([]byte, 1)
	_, err = io.ReadFull(r, end) // frame-end byte
	return typ, channel, payload, err
}

// amqpMethodPayload формирует payload метода: classID + methodID + тело.
func amqpMethodPayload(classID, methodID uint16, body []byte) []byte {
	buf := make([]byte, 4+len(body))
	binary.BigEndian.PutUint16(buf[0:], classID)
	binary.BigEndian.PutUint16(buf[2:], methodID)
	copy(buf[4:], body)
	return buf
}

// amqpLongstr кодирует длинную строку AMQP.
func amqpLongstr(s string) []byte {
	b := []byte(s)
	buf := make([]byte, 4+len(b))
	binary.BigEndian.PutUint32(buf[0:], uint32(len(b))) //nolint:gosec // G115: длина строки в uint32
	copy(buf[4:], b)
	return buf
}

// amqpShortstr кодирует короткую строку AMQP.
func amqpShortstr(s string) []byte {
	b := []byte(s)
	return append([]byte{byte(len(b))}, b...) //nolint:gosec // G115: короткая строка AMQP ≤ 255 байт
}

// sendConnectionStart отправляет Connection.Start (class=10, method=10).
func sendConnectionStart(w io.Writer) error {
	body := make([]byte, 0, 16)
	body = append(body, 0, 9)       // VersionMajor=0, VersionMinor=9
	body = append(body, 0, 0, 0, 0) // empty server properties (table)
	body = append(body, amqpLongstr("PLAIN")...)
	body = append(body, amqpLongstr("en_US")...)
	return writeAMQPFrame(w, 1, 0, amqpMethodPayload(10, 10, body))
}

// sendConnectionTune отправляет Connection.Tune (class=10, method=30).
func sendConnectionTune(w io.Writer) error {
	body := []byte{
		0x08, 0x00, // ChannelMax=2048
		0x00, 0x02, 0x00, 0x00, // FrameMax=131072
		0x00, 0x3C, // Heartbeat=60
	}
	return writeAMQPFrame(w, 1, 0, amqpMethodPayload(10, 30, body))
}

// sendConnectionOpenOk отправляет Connection.OpenOk (class=10, method=41).
func sendConnectionOpenOk(w io.Writer) error {
	body := amqpShortstr("") // reserved1
	return writeAMQPFrame(w, 1, 0, amqpMethodPayload(10, 41, body))
}

// sendChannelOpenOk отправляет Channel.OpenOk (class=20, method=11) для канала ch.
func sendChannelOpenOk(w io.Writer, ch uint16) error {
	body := amqpLongstr("") // reserved1 (longstring в ответе)
	return writeAMQPFrame(w, 1, ch, amqpMethodPayload(20, 11, body))
}

// sendExchangeDeclareOk отправляет Exchange.DeclareOk (class=40, method=11).
func sendExchangeDeclareOk(w io.Writer, ch uint16) error {
	return writeAMQPFrame(w, 1, ch, amqpMethodPayload(40, 11, nil))
}

// sendQueueDeclareOk отправляет Queue.DeclareOk (class=50, method=11).
func sendQueueDeclareOk(w io.Writer, ch uint16, queueName string) error {
	body := amqpShortstr(queueName)
	body = append(body, 0, 0, 0, 0) // MessageCount
	body = append(body, 0, 0, 0, 0) // ConsumerCount
	return writeAMQPFrame(w, 1, ch, amqpMethodPayload(50, 11, body))
}

// sendQueueBindOk отправляет Queue.BindOk (class=50, method=21).
func sendQueueBindOk(w io.Writer, ch uint16) error {
	return writeAMQPFrame(w, 1, ch, amqpMethodPayload(50, 21, nil))
}

// sendBasicQosOk отправляет Basic.QosOk (class=60, method=11).
func sendBasicQosOk(w io.Writer, ch uint16) error {
	return writeAMQPFrame(w, 1, ch, amqpMethodPayload(60, 11, nil))
}

// sendBasicConsumeOk отправляет Basic.ConsumeOk (class=60, method=21).
func sendBasicConsumeOk(w io.Writer, ch uint16) error {
	body := amqpShortstr("test-consumer")
	return writeAMQPFrame(w, 1, ch, amqpMethodPayload(60, 21, body))
}

// sendBasicCancelOk отправляет Basic.CancelOk (class=60, method=31).
func sendBasicCancelOk(w io.Writer, ch uint16) error {
	body := amqpShortstr("test-consumer")
	return writeAMQPFrame(w, 1, ch, amqpMethodPayload(60, 31, body))
}

// doAMQPHandshake выполняет минимальный AMQP-хендшейк на стороне сервера:
// читает заголовок "AMQP\x00\x00\x09\x01", обменивается Connection.Start/TuneOk/Open.
// Возвращает буферизованный writer.
func doAMQPHandshake(conn net.Conn) (*bufio.Writer, error) {
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)

	// Читаем протокольный заголовок клиента
	header := make([]byte, 8)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	// Connection.Start → клиент
	if err := sendConnectionStart(w); err != nil {
		return nil, err
	}
	if err := w.Flush(); err != nil {
		return nil, err
	}

	// Читаем Connection.StartOk от клиента (игнорируем содержимое)
	if _, _, _, err := readAMQPFrame(r); err != nil {
		return nil, err
	}

	// Connection.Tune → клиент
	if err := sendConnectionTune(w); err != nil {
		return nil, err
	}
	if err := w.Flush(); err != nil {
		return nil, err
	}

	// Читаем Connection.TuneOk от клиента
	if _, _, _, err := readAMQPFrame(r); err != nil {
		return nil, err
	}

	// Читаем Connection.Open от клиента
	if _, _, _, err := readAMQPFrame(r); err != nil {
		return nil, err
	}

	// Connection.OpenOk → клиент
	if err := sendConnectionOpenOk(w); err != nil {
		return nil, err
	}
	if err := w.Flush(); err != nil {
		return nil, err
	}

	return w, nil
}

// dialFakeAMQP создаёт локальный TCP-сервер, выполняет AMQP-хендшейк и возвращает
// *amqp.Connection для использования в тестах.
// serverFunc вызывается в горутине после установки соединения — позволяет управлять
// поведением сервера после хендшейка (открытие каналов, объявление очередей и т.д.).
// После завершения serverFunc соединение на стороне сервера закрывается.
func dialFakeAMQP(t *testing.T, serverFunc func(net.Conn, *bufio.Writer, io.Reader)) *amqp.Connection {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Logf("listener close: %v", err)
		}
	})

	connCh := make(chan *amqp.Connection, 1)
	errCh := make(chan error, 1)

	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr != nil {
			errCh <- acceptErr
			return
		}

		w, hsErr := doAMQPHandshake(conn)
		if hsErr != nil {
			closeNetConn(t, conn)
			errCh <- hsErr
			return
		}

		if serverFunc != nil {
			serverFunc(conn, w, bufio.NewReader(conn))
		}

		closeNetConn(t, conn)
	}()

	amqpConn, err := amqp.Dial("amqp://guest:guest@" + ln.Addr().String() + "/")
	if err != nil {
		// Проверяем, не было ли ошибки в горутине сервера
		select {
		case srvErr := <-errCh:
			t.Logf("fake AMQP server error: %v", srvErr)
		default:
		}
		t.Fatalf("failed to dial fake AMQP: %v", err)
	}

	t.Cleanup(func() {
		if err := amqpConn.Close(); err != nil {
			t.Logf("amqpConn close: %v", err)
		}
	})

	connCh <- amqpConn
	return amqpConn
}

// handleFullStartFlow обрабатывает полный flow метода Start:
// Channel.Open → Exchange.Declare → Queue.Declare → Queue.Bind → Basic.Qos → Basic.Consume.
func handleFullStartFlow(t *testing.T, w *bufio.Writer, r io.Reader) {
	t.Helper()

	// Шаги для каждой команды: читаем запрос, отправляем OK.
	steps := []struct {
		name   string
		sendOK func() error
	}{
		{"channel.open", func() error { return sendChannelOpenOk(w, 1) }},
		{"exchange.declare", func() error { return sendExchangeDeclareOk(w, 1) }},
		{"queue.declare", func() error { return sendQueueDeclareOk(w, 1, "integration_service_webhooks") }},
		{"queue.bind", func() error { return sendQueueBindOk(w, 1) }},
		{"basic.qos", func() error { return sendBasicQosOk(w, 1) }},
		{"basic.consume", func() error { return sendBasicConsumeOk(w, 1) }},
	}

	for _, step := range steps {
		if _, _, _, err := readAMQPFrame(r); err != nil {
			t.Logf("fake server: error reading %s frame: %v", step.name, err)
			return
		}
		if err := step.sendOK(); err != nil {
			t.Logf("fake server: error sending %s OK: %v", step.name, err)
			return
		}
		if err := w.Flush(); err != nil {
			t.Logf("fake server: error flushing after %s: %v", step.name, err)
			return
		}
	}
}

// emptyDeliveryRepository заглушка для interfaces.WebhookDeliveryRepository.
type emptyDeliveryRepository struct{}

func (m *emptyDeliveryRepository) Create(_ context.Context, _ *model.WebhookDeliveryAttempt) error {
	return nil
}
func (m *emptyDeliveryRepository) GetByID(_ context.Context, _ uuid.UUID) (*model.WebhookDeliveryAttempt, error) {
	return nil, model.ErrWebhookNotFound
}
func (m *emptyDeliveryRepository) ListByWebhookID(_ context.Context, _ uuid.UUID, _, _ int) ([]*model.WebhookDeliveryAttempt, error) {
	return nil, nil
}
func (m *emptyDeliveryRepository) ListPendingForRetry(_ context.Context, _ int) ([]*model.WebhookDeliveryAttempt, error) {
	return nil, nil
}
func (m *emptyDeliveryRepository) Update(_ context.Context, _ *model.WebhookDeliveryAttempt) error {
	return nil
}
func (m *emptyDeliveryRepository) DeleteOldAttempts(_ context.Context, _ int) (int64, error) {
	return 0, nil
}

// emptyWebhookRepository возвращает пустой список активных webhooks.
type emptyWebhookRepository struct{}

func (m *emptyWebhookRepository) Create(_ context.Context, _ *model.WebhookIntegration) error {
	return nil
}
func (m *emptyWebhookRepository) GetByID(_ context.Context, _ uuid.UUID) (*model.WebhookIntegration, error) {
	return nil, model.ErrWebhookNotFound
}
func (m *emptyWebhookRepository) GetByUserIDAndName(_ context.Context, _ uuid.UUID, _ string) (*model.WebhookIntegration, error) {
	return nil, model.ErrWebhookNotFound
}
func (m *emptyWebhookRepository) ListByUserID(_ context.Context, _ uuid.UUID, _, _ int) ([]*model.WebhookIntegration, error) {
	return nil, nil
}
func (m *emptyWebhookRepository) ListActiveByUserID(_ context.Context, _ uuid.UUID) ([]*model.WebhookIntegration, error) {
	return nil, nil // Return empty list so ProcessAlertEvent returns early
}
func (m *emptyWebhookRepository) Update(_ context.Context, _ *model.WebhookIntegration) error {
	return nil
}
func (m *emptyWebhookRepository) UpdateStats(_ context.Context, _ uuid.UUID, _ *interfaces.WebhookStats) error {
	return nil
}
func (m *emptyWebhookRepository) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *emptyWebhookRepository) CountByUserID(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (m *emptyWebhookRepository) ExistsByName(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}

func newTestDeliverySvc(t *testing.T) *webhook.WebhookDeliveryService {
	t.Helper()
	repo := &emptyWebhookRepository{}
	deliveryRepo := &emptyDeliveryRepository{}
	svc := webhook.NewWebhookDeliveryService(repo, deliveryRepo, nil, &webhook.WebhookDeliveryConfig{
		MaxWorkers:         2,
		WorkerQueueSize:    10,
		MaxRetries:         0,
		RetryIntervals:     []time.Duration{100 * time.Millisecond},
		RetryCheckInterval: 1 * time.Second,
		HTTPTimeout:        100 * time.Millisecond,
	})
	ctx := context.Background()
	require.NoError(t, svc.Start(ctx))
	stopDeliveryService(t, svc)
	return svc
}

// mockAcknowledger имитирует интерфейс amqp.Acknowledger.
type mockAcknowledger struct {
	acked  bool
	nacked bool
}

func (m *mockAcknowledger) Ack(_ uint64, _ bool) error {
	m.acked = true
	return nil
}

func (m *mockAcknowledger) Nack(_ uint64, _ bool, _ bool) error {
	m.nacked = true
	return nil
}

func (m *mockAcknowledger) Reject(_ uint64, _ bool) error {
	return nil
}

func TestNewAlertConsumer(t *testing.T) {
	t.Parallel()

	t.Run("with nil config uses defaults", func(t *testing.T) {
		t.Parallel()
		consumer := NewAlertConsumer(nil, nil, nil)
		require.NotNil(t, consumer)
		assert.Equal(t, "monitor", consumer.exchangeName)
		assert.Equal(t, "alert.triggered", consumer.routingKey)
	})

	t.Run("with custom config", func(t *testing.T) {
		t.Parallel()
		cfg := &AlertConsumerConfig{
			ExchangeName: "custom_exchange",
			RoutingKey:   "custom.key",
			QueueName:    "custom_queue",
		}
		consumer := NewAlertConsumer(nil, nil, cfg)
		require.NotNil(t, consumer)
		assert.Equal(t, "custom_exchange", consumer.exchangeName)
		assert.Equal(t, "custom.key", consumer.routingKey)
	})
}

func TestProcessMessageInvalidJSON(t *testing.T) {
	t.Parallel()

	consumer := NewAlertConsumer(nil, nil, nil)
	ack := &mockAcknowledger{}

	msg := amqp.Delivery{
		Acknowledger: ack,
		Body:         []byte("not-valid-json"),
	}

	consumer.processMessage(context.Background(), msg)
	assert.True(t, ack.nacked, "should nack invalid message")
}

func TestConsumeMessagesContextCancel(t *testing.T) {
	t.Parallel()

	consumer := NewAlertConsumer(nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	msgs := make(chan amqp.Delivery)

	done := make(chan struct{})
	go func() {
		consumer.consumeMessages(ctx, msgs)
		close(done)
	}()

	// Cancel context to stop the consumer
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Error("consumeMessages did not stop after context cancel")
	}
}

func TestProcessMessageValidJSON(t *testing.T) {
	t.Parallel()

	deliverySvc := newTestDeliverySvc(t)
	consumer := NewAlertConsumer(nil, deliverySvc, nil)

	ack := &mockAcknowledger{}

	// Use the AlertEvent struct directly for correct JSON serialization
	alertID := uuid.New()
	monitorID := uuid.New()
	userID := uuid.New()
	alertEvent := struct {
		AlertID     uuid.UUID `json:"AlertID"`
		MonitorID   uuid.UUID `json:"MonitorID"`
		UserID      uuid.UUID `json:"UserID"`
		Severity    string    `json:"Severity"`
		TriggeredAt time.Time `json:"TriggeredAt"`
	}{
		AlertID:     alertID,
		MonitorID:   monitorID,
		UserID:      userID,
		Severity:    "critical",
		TriggeredAt: time.Now(),
	}
	body, err := json.Marshal(alertEvent)
	require.NoError(t, err)

	msg := amqp.Delivery{
		Acknowledger: ack,
		Body:         body,
	}

	consumer.processMessage(context.Background(), msg)
	assert.True(t, ack.acked, "should ack valid message with no webhooks")
}

// errWebhookRepository возвращает ошибку из ListActiveByUserID.
type errWebhookRepository struct {
	emptyWebhookRepository
}

func (m *errWebhookRepository) ListActiveByUserID(_ context.Context, _ uuid.UUID) ([]*model.WebhookIntegration, error) {
	return nil, assert.AnError
}

func TestProcessMessageProcessAlertEventError(t *testing.T) {
	t.Parallel()

	// Используем репозиторий, который возвращает ошибку из ListActiveByUserID
	repo := &errWebhookRepository{}
	deliveryRepo := &emptyDeliveryRepository{}
	svc := webhook.NewWebhookDeliveryService(repo, deliveryRepo, nil, &webhook.WebhookDeliveryConfig{
		MaxWorkers:         1,
		WorkerQueueSize:    10,
		MaxRetries:         0,
		RetryIntervals:     []time.Duration{100 * time.Millisecond},
		RetryCheckInterval: 1 * time.Second,
		HTTPTimeout:        100 * time.Millisecond,
	})
	ctx := context.Background()
	require.NoError(t, svc.Start(ctx))
	stopDeliveryService(t, svc)

	consumer := NewAlertConsumer(nil, svc, nil)
	ack := &mockAcknowledger{}

	alertEvent := struct {
		AlertID     uuid.UUID `json:"AlertID"`
		MonitorID   uuid.UUID `json:"MonitorID"`
		UserID      uuid.UUID `json:"UserID"`
		Severity    string    `json:"Severity"`
		TriggeredAt time.Time `json:"TriggeredAt"`
	}{
		AlertID:     uuid.New(),
		MonitorID:   uuid.New(),
		UserID:      uuid.New(),
		Severity:    "critical",
		TriggeredAt: time.Now(),
	}
	body, err := json.Marshal(alertEvent)
	require.NoError(t, err)

	msg := amqp.Delivery{
		Acknowledger: ack,
		Body:         body,
	}

	consumer.processMessage(context.Background(), msg)
	assert.True(t, ack.nacked, "should nack when ProcessAlertEvent returns error")
}

func TestConsumeMessagesChannelClosed(t *testing.T) {
	t.Parallel()

	consumer := NewAlertConsumer(nil, nil, nil)

	ctx := context.Background()
	msgs := make(chan amqp.Delivery)

	done := make(chan struct{})
	go func() {
		consumer.consumeMessages(ctx, msgs)
		close(done)
	}()

	// Close the channel to simulate AMQP channel closure
	close(msgs)

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Error("consumeMessages did not stop after channel close")
	}
}

// TestConsumeMessagesDelivery проверяет, что consumeMessages передаёт сообщение в processMessage.
func TestConsumeMessagesDelivery(t *testing.T) {
	t.Parallel()

	deliverySvc := newTestDeliverySvc(t)
	consumer := NewAlertConsumer(nil, deliverySvc, nil)

	ctx, cancel := context.WithCancel(context.Background())
	msgs := make(chan amqp.Delivery, 1)

	done := make(chan struct{})
	go func() {
		consumer.consumeMessages(ctx, msgs)
		close(done)
	}()

	ack := &mockAcknowledger{}
	alertEvent := struct {
		AlertID     uuid.UUID `json:"AlertID"`
		MonitorID   uuid.UUID `json:"MonitorID"`
		UserID      uuid.UUID `json:"UserID"`
		Severity    string    `json:"Severity"`
		TriggeredAt time.Time `json:"TriggeredAt"`
	}{
		AlertID:     uuid.New(),
		MonitorID:   uuid.New(),
		UserID:      uuid.New(),
		Severity:    "warning",
		TriggeredAt: time.Now(),
	}
	body, err := json.Marshal(alertEvent)
	require.NoError(t, err)

	msgs <- amqp.Delivery{Acknowledger: ack, Body: body}

	// Ждём обработки
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("consumeMessages did not stop")
	}

	assert.True(t, ack.acked, "delivered message should be acked")
}

// TestAlertConsumerStartChannelError проверяет, что Start возвращает ошибку при сбое Channel().
func TestAlertConsumerStartChannelError(t *testing.T) {
	// Создаём соединение и сразу закрываем его — Channel() вернёт amqp.ErrClosed.
	amqpConn := dialFakeAMQP(t, nil)
	if err := amqpConn.Close(); err != nil {
		t.Logf("amqpConn close: %v", err)
	}

	// Ждём, пока соединение пометится как закрытое
	time.Sleep(20 * time.Millisecond)

	consumer := NewAlertConsumer(amqpConn, nil, nil)
	err := consumer.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create channel")
}

// TestAlertConsumerStartExchangeDeclareError проверяет ошибку при объявлении exchange.
func TestAlertConsumerStartExchangeDeclareError(t *testing.T) {
	// Фейковый сервер: открывает канал, затем закрывает соединение до ExchangeDeclare.
	amqpConn := dialFakeAMQP(t, func(conn net.Conn, w *bufio.Writer, r io.Reader) {
		// Читаем Channel.Open, отправляем Channel.OpenOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendChannelOpenOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Закрываем соединение — клиент получит ошибку при ExchangeDeclare
		closeNetConn(t, conn)
	})

	consumer := NewAlertConsumer(amqpConn, nil, nil)
	err := consumer.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to declare exchange")
}

// TestAlertConsumerStartQueueDeclareError проверяет ошибку при объявлении очереди.
func TestAlertConsumerStartQueueDeclareError(t *testing.T) {
	amqpConn := dialFakeAMQP(t, func(conn net.Conn, w *bufio.Writer, r io.Reader) {
		// Channel.Open → Channel.OpenOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendChannelOpenOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Exchange.Declare → Exchange.DeclareOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendExchangeDeclareOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Закрываем до Queue.Declare
		closeNetConn(t, conn)
	})

	consumer := NewAlertConsumer(amqpConn, nil, nil)
	err := consumer.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to declare queue")
}

// TestAlertConsumerStartQueueBindError проверяет ошибку при привязке очереди.
func TestAlertConsumerStartQueueBindError(t *testing.T) {
	amqpConn := dialFakeAMQP(t, func(conn net.Conn, w *bufio.Writer, r io.Reader) {
		// Channel.Open → OpenOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendChannelOpenOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Exchange.Declare → DeclareOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendExchangeDeclareOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Queue.Declare → DeclareOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendQueueDeclareOk(w, 1, "integration_service_webhooks"); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Закрываем до Queue.Bind
		closeNetConn(t, conn)
	})

	consumer := NewAlertConsumer(amqpConn, nil, nil)
	err := consumer.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to bind queue")
}

// TestAlertConsumerStartQosError проверяет ошибку при установке QoS.
func TestAlertConsumerStartQosError(t *testing.T) {
	amqpConn := dialFakeAMQP(t, func(conn net.Conn, w *bufio.Writer, r io.Reader) {
		// Channel.Open → OpenOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendChannelOpenOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Exchange.Declare → DeclareOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendExchangeDeclareOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Queue.Declare → DeclareOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendQueueDeclareOk(w, 1, "integration_service_webhooks"); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Queue.Bind → BindOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendQueueBindOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Закрываем до Basic.Qos
		closeNetConn(t, conn)
	})

	consumer := NewAlertConsumer(amqpConn, nil, nil)
	err := consumer.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set QoS")
}

// TestAlertConsumerStartConsumeError проверяет ошибку при запуске Basic.Consume.
func TestAlertConsumerStartConsumeError(t *testing.T) {
	amqpConn := dialFakeAMQP(t, func(conn net.Conn, w *bufio.Writer, r io.Reader) {
		// Channel.Open → OpenOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendChannelOpenOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Exchange.Declare → DeclareOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendExchangeDeclareOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Queue.Declare → DeclareOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendQueueDeclareOk(w, 1, "integration_service_webhooks"); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Queue.Bind → BindOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendQueueBindOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Basic.Qos → QosOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendBasicQosOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}
		// Закрываем до Basic.Consume
		closeNetConn(t, conn)
	})

	consumer := NewAlertConsumer(amqpConn, nil, nil)
	err := consumer.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start consuming")
}

// TestAlertConsumerStartSuccess проверяет успешный запуск потребителя.
func TestAlertConsumerStartSuccess(t *testing.T) {
	deliverySvc := newTestDeliverySvc(t)

	// Сервер обрабатывает полный flow Start, затем держит соединение открытым.
	amqpConn := dialFakeAMQP(t, func(conn net.Conn, w *bufio.Writer, r io.Reader) {
		handleFullStartFlow(t, w, r)
		// Держим соединение открытым, пока тест не завершит Consumer
		time.Sleep(500 * time.Millisecond)
	})

	consumer := NewAlertConsumer(amqpConn, deliverySvc, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)

	err := consumer.Start(ctx)
	require.NoError(t, err)
}

// TestAlertConsumerStop проверяет остановку потребителя после успешного запуска.
func TestAlertConsumerStop(t *testing.T) {
	deliverySvc := newTestDeliverySvc(t)

	cancelCh := make(chan struct{})

	amqpConn := dialFakeAMQP(t, func(conn net.Conn, w *bufio.Writer, r io.Reader) {
		handleFullStartFlow(t, w, r)

		// Ждём Basic.Cancel от Stop(), отвечаем Basic.CancelOk
		if _, _, _, err := readAMQPFrame(r); err != nil {
			return
		}
		if err := sendBasicCancelOk(w, 1); err != nil {
			return
		}
		if err := w.Flush(); err != nil {
			return
		}

		// Закрываем соединение
		close(cancelCh)
		closeNetConn(t, conn)
	})

	consumer := NewAlertConsumer(amqpConn, deliverySvc, nil)
	ctx, cancel := context.WithCancel(context.Background())

	err := consumer.Start(ctx)
	require.NoError(t, err)

	// Отменяем контекст — Stop() ожидает ctx.Done()
	cancel()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(stopCancel)

	err = consumer.Stop(stopCtx)
	assert.NoError(t, err)
}
