package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type HandlerFunc func(routingKey string, body []byte) error

type Client struct {
	conn     *amqp.Connection
	exchange string
}

func NewClient(amqpURL, exchange string) (*Client, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, exchange: exchange}, nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) Publish(ctx context.Context, routingKey string, payload any) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(
		c.exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(ctx, c.exchange, routingKey, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
		Timestamp:   time.Now().UTC(),
		Type:        routingKey,
	})
}

func (c *Client) Subscribe(ctx context.Context, queueName string, routingKeys []string, handler HandlerFunc) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}

	if err := ch.ExchangeDeclare(
		c.exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		_ = ch.Close()
		return err
	}

	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		_ = ch.Close()
		return err
	}

	for _, key := range routingKeys {
		if err := ch.QueueBind(q.Name, key, c.exchange, false, nil); err != nil {
			_ = ch.Close()
			return err
		}
	}

	if err := ch.Qos(5, 0, false); err != nil {
		_ = ch.Close()
		return err
	}

	deliveries, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return err
	}

	go func() {
		defer ch.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-deliveries:
				if !ok {
					return
				}
				if err := handler(d.RoutingKey, d.Body); err != nil {
					log.Printf("queue handler failed for %s: %v", d.RoutingKey, err)
					_ = d.Nack(false, false)
					continue
				}
				if err := d.Ack(false); err != nil {
					log.Printf("ack failed: %v", err)
				}
			}
		}
	}()

	return nil
}

func Decode[T any](body []byte, out *T) error {
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode event: %w", err)
	}
	return nil
}
