package bootstrap

import (
	"context"
	"controlplane/infra/psql"
	infraredis "controlplane/infra/redis"
	"controlplane/infra/victoriadb"
	"controlplane/internal/config"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Infra holds all infrastructure clients, initialized once at startup.
type Infra struct {
	DB         *pgxpool.Pool
	Redis      *infraredis.Client
	VictoriaDB *victoriadb.Client
}

// global singleton — set during InitInfra, closed during CloseInfra
var infra *Infra

// InitInfra initializes all infrastructure clients in order.
// Fails fast on any error.
func InitInfra(ctx context.Context, cfg *config.Config) (*Infra, error) {
	// 1. PostgreSQL
	db, err := psql.NewPostgres(ctx, &cfg.Psql)
	if err != nil {
		return nil, fmt.Errorf("bootstrap: psql init failed: %w", err)
	}

	// 2. Redis (cache + stream)
	rds, err := infraredis.NewRedis(ctx, &cfg.Redis)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("bootstrap: redis init failed: %w", err)
	}

	// 3. VictoriaDB
	vdb, err := victoriadb.NewVictoriaDB(ctx, &cfg.VictoriaDB)
	if err != nil {
		db.Close()
		_ = rds.Close()
		return nil, fmt.Errorf("bootstrap: victoriadb init failed: %w", err)
	}

	infra = &Infra{
		DB:         db,
		Redis:      rds,
		VictoriaDB: vdb,
	}

	return infra, nil
}

// CloseInfra closes all infrastructure connections.
func CloseInfra() {
	if infra == nil {
		return
	}
	infra.DB.Close()
	_ = infra.Redis.Close()
	infra.VictoriaDB.Close()
}
