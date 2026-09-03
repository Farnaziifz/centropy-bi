// Package alefgym is the read-only link back to the AlefGym production
// database — the source of truth for customers, orders and course expiry
// that customer sync and segmentation are computed from (see
// loyalty-club-roadmap.html). Every query in this package is a SELECT;
// nothing here ever writes to AlefGym.
package alefgym

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
)

// NewClient opens a connection pool to the AlefGym database. It's plain
// database/sql rather than ent because this service doesn't own that
// schema — mapping AlefGym's ~40 tables into ent models for a handful of
// read queries would be a large maintenance burden for no benefit; a
// handful of hand-written SQL queries against the tables this service
// actually reads is simpler and makes exactly what's being read explicit.
func NewClient(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open alefgym db: %w", err)
	}
	// Small, read-only pool — this service is a light consumer of a
	// production database it doesn't own.
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping alefgym db: %w", err)
	}
	return db, nil
}
