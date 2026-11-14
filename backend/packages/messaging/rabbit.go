// Package messaging implements confirmed publishing and bounded at-least-once delivery.
package messaging

import (
	"context"
	"encoding/json"
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

type Event struct {
	Version     int    `json:"version"`
	ID          string `json:"event_id"`
	UserID      string `json:"user_id"`
	OperationID string `json:"operation_id"`
}
type Bus struct {
	conn     *amqp.Connection
	pub, sub *amqp.Channel
	Queue    string
}

func Open(url, queue string) (*Bus, error) {
	conn, e := amqp.DialConfig(url, amqp.Config{Heartbeat: 10 * time.Second, Dial: amqp.DefaultDial(5 * time.Second)})
	if e != nil {
		return nil, e
	}
	b := &Bus{conn: conn, Queue: queue}
	fail := func(e error) (*Bus, error) { b.Close(); return nil, e }
	b.pub, e = conn.Channel()
	if e != nil {
		return fail(e)
	}
	b.sub, e = conn.Channel()
	if e != nil {
		return fail(e)
	}
	if e = b.pub.ExchangeDeclare("saya.events.v1", "direct", true, false, false, false, nil); e != nil {
		return fail(e)
	}
	if e = b.pub.ExchangeDeclare("saya.dead.v1", "direct", true, false, false, false, nil); e != nil {
		return fail(e)
	}
	if _, e = b.pub.QueueDeclare(queue+".dlq", true, false, false, false, nil); e != nil {
		return fail(e)
	}
	if e = b.pub.QueueBind(queue+".dlq", queue, "saya.dead.v1", false, nil); e != nil {
		return fail(e)
	}
	if _, e = b.pub.QueueDeclare(queue, true, false, false, false, amqp.Table{"x-dead-letter-exchange": "saya.dead.v1", "x-dead-letter-routing-key": queue}); e != nil {
		return fail(e)
	}
	if e = b.pub.QueueBind(queue, queue, "saya.events.v1", false, nil); e != nil {
		return fail(e)
	}
	if e = b.pub.Confirm(false); e != nil {
		return fail(e)
	}
	if e = b.sub.Qos(1, 0, false); e != nil {
		return fail(e)
	}
	return b, nil
}
func (b *Bus) Close() {
	if b.pub != nil {
		b.pub.Close()
	}
	if b.sub != nil {
		b.sub.Close()
	}
	if b.conn != nil {
		b.conn.Close()
	}
}
func (b *Bus) publish(ctx context.Context, body []byte, headers amqp.Table) error {
	confirmation, e := b.pub.PublishWithDeferredConfirmWithContext(ctx, "saya.events.v1", b.Queue, false, false, amqp.Publishing{DeliveryMode: amqp.Persistent, ContentType: "application/json", Body: body, Headers: headers})
	if e != nil {
		return e
	}
	ok, e := confirmation.WaitContext(ctx)
	if e != nil {
		return e
	}
	if !ok {
		return errors.New("broker rejected publish")
	}
	return nil
}
func (b *Bus) Publish(ctx context.Context, event Event) error {
	body, e := json.Marshal(event)
	if e != nil {
		return e
	}
	return b.publish(ctx, body, amqp.Table{"attempt": int32(0)})
}

// Consume uses separate publish and consume channels. Caller supplies durable idempotency.
func (b *Bus) Consume(ctx context.Context, handler func(context.Context, Event) error) error {
	deliveries, e := b.sub.ConsumeWithContext(ctx, b.Queue, "", false, false, false, false, nil)
	if e != nil {
		return e
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return errors.New("delivery channel closed")
			}
			var event Event
			if json.Unmarshal(d.Body, &event) != nil || event.Version != 1 || event.ID == "" || event.UserID == "" || event.OperationID == "" {
				if e = d.Reject(false); e != nil {
					return e
				}
				continue
			}
			task, cancel := context.WithTimeout(ctx, 60*time.Second)
			e = handler(task, event)
			cancel()
			if e == nil {
				if e = d.Ack(false); e != nil {
					return e
				}
				continue
			}
			attempt, _ := d.Headers["attempt"].(int32)
			if attempt >= 2 {
				if e = d.Reject(false); e != nil {
					return e
				}
				continue
			}
			timer := time.NewTimer(time.Duration(attempt+1) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
			}
			retry, cancel := context.WithTimeout(ctx, 5*time.Second)
			e = b.publish(retry, d.Body, amqp.Table{"attempt": attempt + 1})
			cancel()
			if e != nil {
				d.Nack(false, true)
				return e
			}
			if e = d.Ack(false); e != nil {
				return e
			}
		}
	}
}

func (b *Bus) IsClosed() bool { return b.conn.IsClosed() || b.pub.IsClosed() }
