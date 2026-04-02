package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"job-finder/shared/events"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type HandlerFunc func(topic string, body []byte, headers map[string]string) error

type Client struct {
	brokers            []string
	clientID           string
	dlqPrefix          string
	writer             *kafka.Writer
	processLatency     metric.Float64Histogram
	errorCount         metric.Int64Counter
	consumerLag        metric.Int64Histogram
	consumerMessageCnt metric.Int64Counter
}

func NewClient(brokersCSV, clientID string) (*Client, error) {
	brokers := parseBrokers(brokersCSV)
	if len(brokers) == 0 {
		return nil, errors.New("kafka brokers are required")
	}
	if strings.TrimSpace(clientID) == "" {
		clientID = "job-finder"
	}

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireAll,
		BatchTimeout:           15 * time.Millisecond,
		BatchSize:              100,
		Async:                  false,
	}

	meter := otel.Meter("job-finder/stream")
	processLatency, _ := meter.Float64Histogram("kafka.processing.latency.seconds")
	errorCount, _ := meter.Int64Counter("kafka.processing.errors")
	consumerLag, _ := meter.Int64Histogram("kafka.consumer.lag")
	consumerMessageCnt, _ := meter.Int64Counter("kafka.consumer.messages")

	return &Client{
		brokers:            brokers,
		clientID:           clientID,
		dlqPrefix:          events.TopicDLQPrefixDefault,
		writer:             writer,
		processLatency:     processLatency,
		errorCount:         errorCount,
		consumerLag:        consumerLag,
		consumerMessageCnt: consumerMessageCnt,
	}, nil
}

func (c *Client) SetDLQPrefix(prefix string) {
	if strings.TrimSpace(prefix) == "" {
		return
	}
	c.dlqPrefix = prefix
}

func (c *Client) Close() error {
	if c.writer == nil {
		return nil
	}
	return c.writer.Close()
}

func (c *Client) Publish(ctx context.Context, topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal kafka payload: %w", err)
	}

	eventID := uuid.NewString()
	headers := map[string]string{
		"event_id":      eventID,
		"produced_at":   time.Now().UTC().Format(time.RFC3339Nano),
		"producer_name": c.clientID,
	}
	return c.PublishRaw(ctx, topic, eventID, headers, body)
}

func (c *Client) PublishRaw(ctx context.Context, topic, key string, headers map[string]string, payload []byte) error {
	if strings.TrimSpace(topic) == "" {
		return errors.New("topic is required")
	}
	if len(payload) == 0 {
		return errors.New("payload is required")
	}
	if strings.TrimSpace(key) == "" {
		key = uuid.NewString()
	}

	message := kafka.Message{
		Topic:   topic,
		Key:     []byte(key),
		Value:   payload,
		Time:    time.Now().UTC(),
		Headers: mapToHeaders(headers),
	}

	if err := c.writer.WriteMessages(ctx, message); err != nil {
		return fmt.Errorf("publish kafka message topic=%s: %w", topic, err)
	}
	return nil
}

func (c *Client) Subscribe(ctx context.Context, groupID string, topics []string, handler HandlerFunc) error {
	if strings.TrimSpace(groupID) == "" {
		return errors.New("consumer group is required")
	}
	if len(topics) == 0 {
		return errors.New("at least one topic is required")
	}
	if handler == nil {
		return errors.New("handler is required")
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.brokers,
		GroupID:        groupID,
		GroupTopics:    topics,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        2 * time.Second,
		CommitInterval: 0,
		StartOffset:    kafka.FirstOffset,
	})

	go c.consumeLoop(ctx, reader, groupID, handler)
	return nil
}

func (c *Client) consumeLoop(ctx context.Context, reader *kafka.Reader, groupID string, handler HandlerFunc) {
	defer reader.Close()

	for {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			log.Printf("kafka fetch failed group=%s err=%v", groupID, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		start := time.Now()
		headers := headersToMap(message.Headers)
		if _, ok := headers["event_id"]; !ok {
			headers["event_id"] = uuid.NewString()
		}
		headers["topic"] = message.Topic
		lag := message.HighWaterMark - message.Offset
		if lag < 0 {
			lag = 0
		}

		err = handler(message.Topic, message.Value, headers)
		c.recordConsumerMetrics(ctx, message.Topic, groupID, lag, start, err)
		if err != nil {
			log.Printf("kafka handler failed group=%s topic=%s err=%v", groupID, message.Topic, err)
			if dlqErr := c.publishDLQ(ctx, message, headers, err); dlqErr != nil {
				log.Printf("kafka dlq publish failed group=%s topic=%s err=%v", groupID, message.Topic, dlqErr)
			}
		}

		if commitErr := reader.CommitMessages(ctx, message); commitErr != nil {
			log.Printf("kafka commit failed group=%s topic=%s err=%v", groupID, message.Topic, commitErr)
		}
	}
}

func (c *Client) publishDLQ(ctx context.Context, message kafka.Message, headers map[string]string, handlerErr error) error {
	dlqHeaders := make(map[string]string, len(headers)+3)
	for key, value := range headers {
		dlqHeaders[key] = value
	}
	dlqHeaders["dlq_error"] = handlerErr.Error()
	dlqHeaders["dlq_original_topic"] = message.Topic
	dlqHeaders["dlq_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	return c.PublishRaw(ctx, events.DLQTopicFor(message.Topic, c.dlqPrefix), string(message.Key), dlqHeaders, message.Value)
}

func (c *Client) recordConsumerMetrics(ctx context.Context, topic, groupID string, lag int64, startedAt time.Time, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("topic", topic),
		attribute.String("consumer_group", groupID),
	}
	c.consumerMessageCnt.Add(ctx, 1, metric.WithAttributes(attrs...))
	c.consumerLag.Record(ctx, lag, metric.WithAttributes(attrs...))
	c.processLatency.Record(ctx, time.Since(startedAt).Seconds(), metric.WithAttributes(attrs...))
	if err != nil {
		c.errorCount.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func parseBrokers(brokersCSV string) []string {
	raw := strings.Split(brokersCSV, ",")
	brokers := make([]string, 0, len(raw))
	for _, broker := range raw {
		candidate := strings.TrimSpace(broker)
		if candidate == "" {
			continue
		}
		brokers = append(brokers, candidate)
	}
	return brokers
}

func mapToHeaders(headers map[string]string) []kafka.Header {
	if len(headers) == 0 {
		return nil
	}
	result := make([]kafka.Header, 0, len(headers))
	for key, value := range headers {
		result = append(result, kafka.Header{Key: key, Value: []byte(value)})
	}
	return result
}

func headersToMap(headers []kafka.Header) map[string]string {
	result := make(map[string]string, len(headers))
	for _, header := range headers {
		result[header.Key] = string(header.Value)
	}
	return result
}

func Decode[T any](body []byte, out *T) error {
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode stream event: %w", err)
	}
	return nil
}
