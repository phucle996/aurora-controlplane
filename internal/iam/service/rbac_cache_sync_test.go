package iam_service

import (
	"context"
	"testing"
	"time"

	"controlplane/internal/http/middleware"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRbacCacheSyncInvalidatesRolesFromPubSubAndHealsOnEpochTick(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	registry := middleware.NewRoleRegistry()
	registry.Set("admin", middleware.RoleEntry{Level: 0})
	registry.Set("viewer", middleware.RoleEntry{Level: 10})

	syncer := NewRbacCacheSync(client, registry)
	if syncer == nil {
		t.Fatal("expected syncer")
	}
	syncer.Start(context.Background())
	t.Cleanup(syncer.Stop)

	bus := NewRedisRbacCacheBus(client)
	if err := bus.PublishInvalidateRole(context.Background(), "admin"); err != nil {
		t.Fatalf("publish role invalidation: %v", err)
	}
	waitForCacheEmptyRole(t, registry, "admin")

	registry.Set("viewer", middleware.RoleEntry{Level: 10})
	if err := bus.PublishInvalidateAll(context.Background()); err != nil {
		t.Fatalf("publish flush-all invalidation: %v", err)
	}
	waitForCacheEmptyRole(t, registry, "viewer")
}

func TestRbacCacheSyncHealsOnMissedEpochTick(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	registry := middleware.NewRoleRegistry()
	registry.Set("admin", middleware.RoleEntry{Level: 0})

	syncer := NewRbacCacheSync(client, registry)
	if syncer == nil {
		t.Fatal("expected syncer")
	}

	syncer.epoch = 1
	if err := client.Set(context.Background(), rbacEpochKey, 2, 0).Err(); err != nil {
		t.Fatalf("seed epoch: %v", err)
	}

	syncer.syncEpoch(context.Background())

	if _, ok := registry.Get("admin"); ok {
		t.Fatalf("expected registry to be flushed when epoch jumps")
	}
}

func waitForCacheEmptyRole(t *testing.T, registry *middleware.RoleRegistry, role string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := registry.Get(role); !ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected role %q to be invalidated", role)
}
