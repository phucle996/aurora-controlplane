package redisprojection

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"controlplane/internal/primitive/leaseassign"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestSinkPublishActive(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx := context.Background()
	sink := NewSink(client, "test:runtime")

	now := time.Now().UTC()
	err := sink.PublishActive(ctx, "smtp:gateway", []leaseassign.Assignment{
		{
			WorkID:          "w-1",
			OwnerNodeID:     "dp-1",
			AssignmentState: leaseassign.StateActive,
			DesiredState:    leaseassign.StateActive,
			Generation:      1,
			LeaseExpiresAt:  now.Add(time.Minute),
		},
		{
			WorkID:          "w-2",
			OwnerNodeID:     "dp-2",
			AssignmentState: leaseassign.StatePending,
			DesiredState:    leaseassign.StateActive,
			Generation:      1,
			LeaseExpiresAt:  now.Add(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("publish active #1: %v", err)
	}

	key1 := "test:runtime:smtp:gateway:w-1"
	raw1, err := mr.Get(key1)
	if err != nil {
		t.Fatalf("expected key %s", key1)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw1), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["owner_node_id"] != "dp-1" {
		t.Fatalf("expected owner dp-1, got %v", payload["owner_node_id"])
	}

	if mr.Exists("test:runtime:smtp:gateway:w-2") {
		t.Fatalf("expected pending row not to be projected")
	}

	err = sink.PublishActive(ctx, "smtp:gateway", []leaseassign.Assignment{
		{
			WorkID:          "w-3",
			OwnerNodeID:     "dp-3",
			AssignmentState: leaseassign.StateActive,
			DesiredState:    leaseassign.StateActive,
			Generation:      2,
			LeaseExpiresAt:  now.Add(time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("publish active #2: %v", err)
	}

	if mr.Exists("test:runtime:smtp:gateway:w-1") {
		t.Fatalf("expected stale active key w-1 to be removed")
	}
	if !mr.Exists("test:runtime:smtp:gateway:w-3") {
		t.Fatalf("expected new active key w-3 to exist")
	}
}

func TestSinkPublishActive_NoClientNoop(t *testing.T) {
	t.Parallel()

	var sink *Sink
	if err := sink.PublishActive(context.Background(), "any", nil); err != nil {
		t.Fatalf("expected nil error for nil sink")
	}
}
