package piano

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"time"

	// libsql (Turso) DB driver.
	_ "github.com/libsql/libsql-client-go/libsql"
)

type Repo struct {
	db *sql.DB
}

func NewDB() (*Repo, error) {
	// Database configuration from deployment.
	dburl := os.Getenv("DATABASE_URL")
	if dburl == "" {
		scheme := os.Getenv("DB_SCHEME")
		host := os.Getenv("DB_HOST")
		token := os.Getenv("DB_TOKEN")
		u := url.URL{
			Scheme:   scheme,
			Host:     host,
			RawQuery: url.Values{"authToken": {token}}.Encode(),
		}
		dburl = u.String()
	}

	db, err := sql.Open("libsql", dburl)
	if err != nil {
		return nil, fmt.Errorf("open DB: %w", err)
	}

	return &Repo{db: db}, nil
}

func (r *Repo) StoreSession(ctx context.Context, at time.Time, length time.Duration) error {
	_, err := r.db.ExecContext(
		ctx,
		`insert into piano_sessions(at, seconds) values (?, ?)`,
		at.Format(time.RFC3339),
		int(length.Seconds()))
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

func (r *Repo) Sessions(ctx context.Context) (map[string]int, error) {
	counts := make(map[string]int)
	rows, err := r.db.QueryContext(ctx, `select at,seconds from piano_sessions`)
	if err != nil {
		return nil, fmt.Errorf("insert: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		// For some reason the column is returned as text, not datetime.
		var atRaw string
		var duration int
		err := rows.Scan(&atRaw, &duration)
		if err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		at, err := time.Parse(time.RFC3339, atRaw)
		if err != nil {
			return nil, fmt.Errorf("parse time: %w", err)
		}
		counts[at.Format("2006-01-02")] += duration
	}

	return counts, nil
}
