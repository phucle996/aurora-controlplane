package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	core_errorx "controlplane/internal/core/errorx"
	"controlplane/internal/virtualmachine/domain/entity"
	vm_errorx "controlplane/internal/virtualmachine/errorx"
	vm_model "controlplane/internal/virtualmachine/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HostRepository struct {
	db *pgxpool.Pool
}

func NewHostRepository(db *pgxpool.Pool) *HostRepository {
	return &HostRepository{db: db}
}

func (r *HostRepository) ListHosts(ctx context.Context, filter entity.HostListFilter) (*entity.HostPage, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("virtual-machine repo: host db is nil")
	}

	page, limit := normalizePagination(filter.Page, filter.Limit)
	offset := (page - 1) * limit
	query := strings.TrimSpace(filter.Query)
	status := strings.ToLower(strings.TrimSpace(filter.Status))
	zoneSlug := strings.TrimSpace(filter.ZoneSlug)

	rows, err := r.db.Query(ctx, `
		SELECT
			h.host_id,
			h.agent_id,
			h.zone_id,
			z.slug AS zone_slug,
			h.data_plane_id,
			h.hostname,
			h.private_ip,
			h.hypervisor_type,
			h.agent_version,
			h.capabilities_json,
			h.cpu_cores,
			h.memory_bytes,
			h.disk_bytes,
			h.status,
			h.last_seen_at,
			h.created_at,
			h.updated_at,
			COUNT(*) OVER() AS total_count
		FROM virtual_machine.hosts h
		JOIN core.zones z ON z.id = h.zone_id
		WHERE ($1 = '' OR h.host_id ILIKE '%' || $1 || '%' OR h.agent_id ILIKE '%' || $1 || '%' OR h.hostname ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR h.status = $2)
		  AND ($3 = '' OR z.slug = $3)
		ORDER BY h.updated_at DESC, h.created_at DESC, h.hostname ASC
		LIMIT $4 OFFSET $5
	`, query, status, zoneSlug, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("virtual-machine repo: list hosts: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.Host, 0, limit)
	var total int64
	for rows.Next() {
		var (
			row        vm_model.Host
			totalCount int64
		)
		if err := rows.Scan(
			&row.HostID,
			&row.AgentID,
			&row.ZoneID,
			&row.ZoneSlug,
			&row.DataPlaneID,
			&row.Hostname,
			&row.PrivateIP,
			&row.HypervisorType,
			&row.AgentVersion,
			&row.CapabilitiesJSON,
			&row.CPUCores,
			&row.MemoryBytes,
			&row.DiskBytes,
			&row.Status,
			&row.LastSeenAt,
			&row.CreatedAt,
			&row.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, fmt.Errorf("virtual-machine repo: scan host: %w", err)
		}
		total = totalCount
		items = append(items, vm_model.HostModelToEntity(&row))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("virtual-machine repo: iterate hosts: %w", err)
	}

	return &entity.HostPage{
		Items: items,
		Pagination: entity.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages(total, limit),
		},
	}, nil
}

func (r *HostRepository) GetHostByID(ctx context.Context, hostID string) (*entity.Host, error) {
	if r == nil || r.db == nil {
		return nil, vm_errorx.ErrHostNotFound
	}

	var row vm_model.Host
	err := r.db.QueryRow(ctx, `
		SELECT
			h.host_id,
			h.agent_id,
			h.zone_id,
			z.slug AS zone_slug,
			h.data_plane_id,
			h.hostname,
			h.private_ip,
			h.hypervisor_type,
			h.agent_version,
			h.capabilities_json,
			h.cpu_cores,
			h.memory_bytes,
			h.disk_bytes,
			h.status,
			h.last_seen_at,
			h.created_at,
			h.updated_at
		FROM virtual_machine.hosts h
		JOIN core.zones z ON z.id = h.zone_id
		WHERE h.host_id = $1
	`, hostID).Scan(
		&row.HostID,
		&row.AgentID,
		&row.ZoneID,
		&row.ZoneSlug,
		&row.DataPlaneID,
		&row.Hostname,
		&row.PrivateIP,
		&row.HypervisorType,
		&row.AgentVersion,
		&row.CapabilitiesJSON,
		&row.CPUCores,
		&row.MemoryBytes,
		&row.DiskBytes,
		&row.Status,
		&row.LastSeenAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, vm_errorx.ErrHostNotFound
		}
		return nil, fmt.Errorf("virtual-machine repo: get host: %w", err)
	}

	return vm_model.HostModelToEntity(&row), nil
}

func (r *HostRepository) GetHostByAgentID(ctx context.Context, agentID string) (*entity.Host, error) {
	if r == nil || r.db == nil {
		return nil, vm_errorx.ErrHostNotFound
	}

	var row vm_model.Host
	err := r.db.QueryRow(ctx, `
		SELECT
			h.host_id,
			h.agent_id,
			h.zone_id,
			z.slug AS zone_slug,
			h.data_plane_id,
			h.hostname,
			h.private_ip,
			h.hypervisor_type,
			h.agent_version,
			h.capabilities_json,
			h.cpu_cores,
			h.memory_bytes,
			h.disk_bytes,
			h.status,
			h.last_seen_at,
			h.created_at,
			h.updated_at
		FROM virtual_machine.hosts h
		JOIN core.zones z ON z.id = h.zone_id
		WHERE h.agent_id = $1
	`, agentID).Scan(
		&row.HostID,
		&row.AgentID,
		&row.ZoneID,
		&row.ZoneSlug,
		&row.DataPlaneID,
		&row.Hostname,
		&row.PrivateIP,
		&row.HypervisorType,
		&row.AgentVersion,
		&row.CapabilitiesJSON,
		&row.CPUCores,
		&row.MemoryBytes,
		&row.DiskBytes,
		&row.Status,
		&row.LastSeenAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, vm_errorx.ErrHostNotFound
		}
		return nil, fmt.Errorf("virtual-machine repo: get host by agent: %w", err)
	}

	return vm_model.HostModelToEntity(&row), nil
}

func (r *HostRepository) UpsertHost(ctx context.Context, host *entity.Host) (*entity.Host, error) {
	if r == nil || r.db == nil || host == nil {
		return nil, vm_errorx.ErrHostInvalid
	}

	zoneID, err := r.resolveZoneID(ctx, host.ZoneSlug)
	if err != nil {
		return nil, err
	}

	var row vm_model.Host
	err = r.db.QueryRow(ctx, `
		INSERT INTO virtual_machine.hosts (
			host_id, agent_id, zone_id, data_plane_id,
			hostname, private_ip, hypervisor_type, agent_version,
			capabilities_json, cpu_cores, memory_bytes, disk_bytes,
			status, last_seen_at, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, NOW(), NOW()
		)
		ON CONFLICT (host_id) DO UPDATE SET
			agent_id = EXCLUDED.agent_id,
			zone_id = EXCLUDED.zone_id,
			data_plane_id = EXCLUDED.data_plane_id,
			hostname = EXCLUDED.hostname,
			private_ip = EXCLUDED.private_ip,
			hypervisor_type = EXCLUDED.hypervisor_type,
			agent_version = EXCLUDED.agent_version,
			capabilities_json = EXCLUDED.capabilities_json,
			cpu_cores = EXCLUDED.cpu_cores,
			memory_bytes = EXCLUDED.memory_bytes,
			disk_bytes = EXCLUDED.disk_bytes,
			status = EXCLUDED.status,
			last_seen_at = EXCLUDED.last_seen_at,
			updated_at = NOW()
		RETURNING
			host_id, agent_id, zone_id,
			(SELECT slug FROM core.zones WHERE id = zone_id) AS zone_slug,
			data_plane_id, hostname, private_ip, hypervisor_type, agent_version,
			capabilities_json, cpu_cores, memory_bytes, disk_bytes, status, last_seen_at, created_at, updated_at
	`, host.HostID, host.AgentID, zoneID, host.DataPlaneID, host.Hostname, host.PrivateIP, host.HypervisorType, host.AgentVersion, host.CapabilitiesJSON, host.CPUCores, host.MemoryBytes, host.DiskBytes, host.Status, host.LastSeenAt).Scan(
		&row.HostID,
		&row.AgentID,
		&row.ZoneID,
		&row.ZoneSlug,
		&row.DataPlaneID,
		&row.Hostname,
		&row.PrivateIP,
		&row.HypervisorType,
		&row.AgentVersion,
		&row.CapabilitiesJSON,
		&row.CPUCores,
		&row.MemoryBytes,
		&row.DiskBytes,
		&row.Status,
		&row.LastSeenAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if hostConflictErr(err) {
			return nil, vm_errorx.ErrHostConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core_errorx.ErrZoneNotFound
		}
		return nil, fmt.Errorf("virtual-machine repo: upsert host: %w", err)
	}

	return vm_model.HostModelToEntity(&row), nil
}

func (r *HostRepository) UpdateHostStatus(ctx context.Context, hostID, dataPlaneID, status string, lastSeenAt time.Time) (*entity.Host, error) {
	if r == nil || r.db == nil {
		return nil, vm_errorx.ErrHostNotFound
	}

	var row vm_model.Host
	err := r.db.QueryRow(ctx, `
		UPDATE virtual_machine.hosts
		SET status = $3,
		    last_seen_at = $4,
		    updated_at = NOW()
		WHERE host_id = $1 AND data_plane_id = $2
		RETURNING
			host_id, agent_id, zone_id,
			(SELECT slug FROM core.zones WHERE id = zone_id) AS zone_slug,
			data_plane_id, hostname, private_ip, hypervisor_type, agent_version,
			capabilities_json, cpu_cores, memory_bytes, disk_bytes, status, last_seen_at, created_at, updated_at
	`, hostID, dataPlaneID, status, lastSeenAt).Scan(
		&row.HostID,
		&row.AgentID,
		&row.ZoneID,
		&row.ZoneSlug,
		&row.DataPlaneID,
		&row.Hostname,
		&row.PrivateIP,
		&row.HypervisorType,
		&row.AgentVersion,
		&row.CapabilitiesJSON,
		&row.CPUCores,
		&row.MemoryBytes,
		&row.DiskBytes,
		&row.Status,
		&row.LastSeenAt,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, vm_errorx.ErrHostNotFound
		}
		return nil, fmt.Errorf("virtual-machine repo: update host status: %w", err)
	}

	return vm_model.HostModelToEntity(&row), nil
}

func (r *HostRepository) ListHostOptions(ctx context.Context, filter entity.HostListFilter) ([]*entity.HostOption, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("virtual-machine repo: host db is nil")
	}

	query := strings.TrimSpace(filter.Query)
	status := strings.ToLower(strings.TrimSpace(filter.Status))
	zoneSlug := strings.TrimSpace(filter.ZoneSlug)

	rows, err := r.db.Query(ctx, `
		SELECT
			h.host_id,
			COALESCE(NULLIF(h.hostname, ''), h.host_id) || ' (' || z.slug || ')' AS label,
			z.slug AS zone_slug,
			h.status,
			h.data_plane_id
		FROM virtual_machine.hosts h
		JOIN core.zones z ON z.id = h.zone_id
		WHERE ($1 = '' OR h.host_id ILIKE '%' || $1 || '%' OR h.hostname ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR h.status = $2)
		  AND ($3 = '' OR z.slug = $3)
		ORDER BY h.hostname ASC, h.created_at ASC
		LIMIT 200
	`, query, status, zoneSlug)
	if err != nil {
		return nil, fmt.Errorf("virtual-machine repo: list host options: %w", err)
	}
	defer rows.Close()

	items := make([]*entity.HostOption, 0)
	for rows.Next() {
		var row vm_model.HostOption
		if err := rows.Scan(&row.HostID, &row.Label, &row.ZoneSlug, &row.Status, &row.DataPlaneID); err != nil {
			return nil, fmt.Errorf("virtual-machine repo: scan host option: %w", err)
		}
		items = append(items, vm_model.HostOptionModelToEntity(&row))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("virtual-machine repo: iterate host options: %w", err)
	}

	return items, nil
}

func (r *HostRepository) resolveZoneID(ctx context.Context, zoneSlug string) (string, error) {
	if strings.TrimSpace(zoneSlug) == "" {
		return "", vm_errorx.ErrHostInvalid
	}

	var zoneID string
	err := r.db.QueryRow(ctx, `SELECT id FROM core.zones WHERE slug = $1`, zoneSlug).Scan(&zoneID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", core_errorx.ErrZoneNotFound
		}
		return "", fmt.Errorf("virtual-machine repo: resolve zone id: %w", err)
	}
	return zoneID, nil
}

func hostConflictErr(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && (pgErr.ConstraintName == "hosts_agent_id_key" || pgErr.ConstraintName == "hosts_pkey")
}

func normalizePagination(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func totalPages(total int64, limit int) int {
	if total <= 0 || limit <= 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}
