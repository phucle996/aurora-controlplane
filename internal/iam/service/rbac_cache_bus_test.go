package iam_service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisRbacCacheBusPublishesRoleInvalidationEvent(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	sub := client.Subscribe(context.Background(), rbacInvalidateChannel)
	t.Cleanup(func() { _ = sub.Close() })
	if _, err := sub.Receive(context.Background()); err != nil {
		t.Fatalf("subscribe ack: %v", err)
	}

	bus := NewRedisRbacCacheBus(client)
	if err := bus.PublishInvalidateRole(context.Background(), "admin"); err != nil {
		t.Fatalf("publish role invalidation: %v", err)
	}

	msg := waitForRedisPubSubMessage(t, sub)
	var event rbacInvalidateEvent
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if event.Kind != rbacInvalidateRole {
		t.Fatalf("expected role event, got %q", event.Kind)
	}
	if event.Role != "admin" {
		t.Fatalf("expected role admin, got %q", event.Role)
	}
	if event.Epoch != 1 {
		t.Fatalf("expected epoch 1, got %d", event.Epoch)
	}

	epoch, err := client.Get(context.Background(), rbacEpochKey).Int64()
	if err != nil {
		t.Fatalf("read epoch: %v", err)
	}
	if epoch != 1 {
		t.Fatalf("expected redis epoch 1, got %d", epoch)
	}
}

func TestRedisRbacCacheBusPublishesFlushAllEvent(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	sub := client.Subscribe(context.Background(), rbacInvalidateChannel)
	t.Cleanup(func() { _ = sub.Close() })
	if _, err := sub.Receive(context.Background()); err != nil {
		t.Fatalf("subscribe ack: %v", err)
	}

	bus := NewRedisRbacCacheBus(client)
	if err := bus.PublishInvalidateAll(context.Background()); err != nil {
		t.Fatalf("publish flush-all invalidation: %v", err)
	}

	msg := waitForRedisPubSubMessage(t, sub)
	var event rbacInvalidateEvent
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if event.Kind != rbacInvalidateAll {
		t.Fatalf("expected flush-all event, got %q", event.Kind)
	}
	if event.Role != "" {
		t.Fatalf("expected flush-all event to omit role, got %q", event.Role)
	}
	if event.Epoch != 1 {
		t.Fatalf("expected epoch 1, got %d", event.Epoch)
	}
}

func waitForRedisPubSubMessage(t *testing.T, sub *redis.PubSub) *redis.Message {
	t.Helper()

	deadline := time.After(2 * time.Second)
	select {
	case msg := <-sub.Channel():
		return msg
	case <-deadline:
		t.Fatal("timed out waiting for redis pubsub message")
		return nil
	}
}
