package messaging

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestIntegrationRetryDLQ(t *testing.T) {
	url := os.Getenv("RABBIT_URL")
	if url == "" {
		t.Skip("RABBIT_URL required")
	}
	queue := fmt.Sprintf("saya.test.%d", time.Now().UnixNano())
	publisher, e := Open(url, queue)
	if e != nil {
		t.Fatal(e)
	}
	defer publisher.Close()
	worker, e := Open(url, queue)
	if e != nil {
		t.Fatal(e)
	}
	defer worker.Close()
	defer func() {
		publisher.pub.QueueDelete(queue, false, false, false)
		publisher.pub.QueueDelete(queue+".dlq", false, false, false)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if e = publisher.Publish(ctx, Event{Version: 1, ID: "test", UserID: "test", OperationID: "test"}); e != nil {
		t.Fatal(e)
	}
	attempts := make(chan struct{}, 3)
	done := make(chan error, 1)
	go func() {
		done <- worker.Consume(ctx, func(context.Context, Event) error { attempts <- struct{}{}; return errors.New("injected failure") })
	}()
	for i := 0; i < 3; i++ {
		select {
		case <-attempts:
		case <-ctx.Done():
			t.Fatal("retry timeout")
		}
	}
	for {
		delivery, ok, e := publisher.pub.Get(queue+".dlq", false)
		if e != nil {
			t.Fatal(e)
		}
		if ok {
			if e = delivery.Ack(false); e != nil {
				t.Fatal(e)
			}
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("DLQ missing")
		case <-time.After(50 * time.Millisecond):
		}
	}
	cancel()
	if e = <-done; e != nil {
		t.Fatal(e)
	}
}
