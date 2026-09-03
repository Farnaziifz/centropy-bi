// Package persistence wires ent to this service's own Postgres database and
// implements every domain repository interface against the generated ent
// client. It has nothing to do with AlefGym — that read-only link lives in
// internal/infrastructure/alefgym.
package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	entschema "entgo.io/ent/dialect/sql/schema"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver

	"centropy-affilate/ent"
	"centropy-affilate/internal/infrastructure/config"
)

// NewEntClient opens a pgx-backed connection pool and wraps it as an ent
// client. pgx's binary wire protocol and statement cache outperform
// lib/pq/database-sql-only drivers under load, which is why it's used here
// instead of the more common "postgres" driver.
func NewEntClient(cfg config.DBConfig) (*ent.Client, error) {
	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConn)
	db.SetMaxIdleConns(cfg.MaxIdleConn)
	db.SetConnMaxLifetime(cfg.ConnMaxLife)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	drv := entsql.OpenDB(dialect.Postgres, db)
	return ent.NewClient(ent.Driver(drv)), nil
}

// RunMigrations applies the ent-derived schema, dropping columns/indexes
// that a schema change removed so the database always matches the current
// schema exactly. Adequate for this service's current size; swap for an
// Atlas-managed migration pipeline before this needs zero-downtime schema
// changes in production.
func RunMigrations(ctx context.Context, client *ent.Client) error {
	return client.Schema.Create(ctx, entschema.WithDropColumn(true), entschema.WithDropIndex(true))
}
