package redisprojection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"controlplane/internal/primitive/leaseassign"

	goredis "github.com/redis/go-redis/v9"
)

type Sink struct {
	client *goredis.Client
	prefix string
}

func NewSink(client *goredis.Client, prefix string) *Sink {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "runtime:assignments"
	}
	return &Sink{
		client: client,
		prefix: prefix,
	}
}

func (s *Sink) Enabled() bool {
	return s != nil && s.client != nil
}

// PublishActive projects active ownership into Redis after DB commit.
// Redis is projection only; failures are returned to caller as non-fatal decision.
func (s *Sink) PublishActive(ctx context.Context, runtimeKey string, rows []leaseassign.Assignment) error {
	if !s.Enabled() || strings.TrimSpace(runtimeKey) == "" {
		return nil
	}

	activeByWork := make(map[string]leaseassign.Assignment)
	for _, row := range rows {
		if row.WorkID == "" || row.AssignmentState != leaseassign.StateActive || row.OwnerNodeID == "" {
			continue
		}
		activeByWork[row.WorkID] = row
	}

	indexKey := s.indexKey(runtimeKey)
	currentWorkIDs, err := s.client.SMembers(ctx, indexKey).Result()
	if err != nil && err != goredis.Nil {
		return fmt.Errorf("projection redis: list current keys: %w", err)
	}

	_, err = s.client.Pipelined(ctx, func(pipe goredis.Pipeliner) error {
		for workID, row := range activeByWork {
			payload, err := json.Marshal(map[string]any{
				"work_id":          row.WorkID,
				"owner_node_id":    row.OwnerNodeID,
				"generation":       row.Generation,
				"assignment_state": row.AssignmentState,
				"desired_state":    row.DesiredState,
				"lease_expires_at": row.LeaseExpiresAt.UTC().Format(time.RFC3339Nano),
				"metadata":         row.Metadata,
			})
			if err != nil {
				return fmt.Errorf("projection redis: marshal row: %w", err)
			}
			key := s.assignmentKey(runtimeKey, workID)
			pipe.Set(ctx, key, payload, 0)
			pipe.SAdd(ctx, indexKey, workID)
		}

		for _, workID := range currentWorkIDs {
			if _, keep := activeByWork[workID]; keep {
				continue
			}
			pipe.Del(ctx, s.assignmentKey(runtimeKey, workID))
			pipe.SRem(ctx, indexKey, workID)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("projection redis: apply projection: %w", err)
	}
	return nil
}

func (s *Sink) assignmentKey(runtimeKey, workID string) string {
	return fmt.Sprintf("%s:%s:%s", s.prefix, runtimeKey, workID)
}

func (s *Sink) indexKey(runtimeKey string) string {
	return fmt.Sprintf("%s:%s:index", s.prefix, runtimeKey)
}
