package piano

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	// Postgres driver.
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Repo struct {
	db *sql.DB
}

func NewDB() (*Repo, error) {
	// Database configuration from deployment.
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("open DB: %w", err)
	}

	db.SetConnMaxIdleTime(3 * time.Minute)
	db.SetMaxIdleConns(3)
	db.SetMaxOpenConns(3)

	return &Repo{db: db}, nil
}

func (r *Repo) StoreSession(ctx context.Context, at time.Time, length time.Duration) error {
	_, err := r.db.ExecContext(ctx,
		`insert into piano_sessions (at, seconds) values ($1, $2)`,
		at, int(length.Seconds()),
	)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

func (r *Repo) Sessions(ctx context.Context) (map[string]int, error) {
	counts := make(map[string]int)
	rows, err := r.db.QueryContext(ctx, `select at, seconds from piano_sessions`)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var at time.Time
		var duration int
		err := rows.Scan(&at, &duration)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		counts[at.Format(time.DateOnly)] += duration
	}

	return counts, rows.Err()
}
