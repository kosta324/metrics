package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kosta324/metrics.git/internal/models"
	"strings"
	"time"
)

type SQLRepo struct {
	db *sql.DB
}

func isRetriablePgError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return strings.HasPrefix(pgErr.Code, pgerrcode.ConnectionException)
	}
	return false
}

func NewSQLStorage(dsn string) (*SQLRepo, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS gauges (
            name TEXT PRIMARY KEY,
            value DOUBLE PRECISION
        );
        CREATE TABLE IF NOT EXISTS counters (
            name TEXT PRIMARY KEY,
            delta BIGINT
        );
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &SQLRepo{db: db}, nil
}

func (r *SQLRepo) DB() *sql.DB {
	return r.db
}

func (r *SQLRepo) Add(metricType, name, value string) error {
	retries := []time.Duration{0, 1 * time.Second, 3 * time.Second, 5 * time.Second}
	var err error
	for attempt, delay := range retries {
		if attempt > 0 {
			time.Sleep(delay)
		}
		switch metricType {
		case "gauge":
			_, err = r.db.Exec(`
				INSERT INTO gauges (name, value)
				VALUES ($1, $2)
				ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
			`, name, value)
		case "counter":
			_, err = r.db.Exec(`
				INSERT INTO counters (name, delta)
				VALUES ($1, $2)
				ON CONFLICT (name) DO UPDATE SET delta = counters.delta + EXCLUDED.delta
			`, name, value)
		default:
			return fmt.Errorf("unsupported metric type: %s", metricType)
		}
		if err == nil || !isRetriablePgError(err) {
			break
		}
	}
	return err
}

func (r *SQLRepo) Get(metricType, name string) (string, error) {
	var result string
	switch metricType {
	case "gauge":
		err := r.db.QueryRow("SELECT value FROM gauges WHERE name = $1", name).Scan(&result)
		return result, err
	case "counter":
		err := r.db.QueryRow("SELECT delta FROM counters WHERE name = $1", name).Scan(&result)
		return result, err
	default:
		return "", fmt.Errorf("unsupported metric type: %s", metricType)
	}
}

func (r *SQLRepo) GetAll() map[string]string {
	result := make(map[string]string)
	rows, err := r.db.Query("SELECT name, value FROM gauges")
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var name, val string
		if err := rows.Scan(&name, &val); err == nil {
			result[name] = val
		}
	}
	if err := rows.Err(); err != nil {
		fmt.Printf("error reading gauges: %v\n", err)
	}

	rows, err = r.db.Query("SELECT name, delta FROM counters")
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var name, val string
		if err := rows.Scan(&name, &val); err == nil {
			result[name] = val
		}
	}
	if err := rows.Err(); err != nil {
		fmt.Printf("error reading counters: %v\n", err)
	}

	return result
}

func (r *SQLRepo) AddBatch(metrics []models.Metrics) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, m := range metrics {
		switch m.MType {
		case "gauge":
			if m.Value == nil {
				return fmt.Errorf("missing gauge value for metric %s", m.ID)
			}
			_, err := tx.Exec(`
				INSERT INTO gauges (name, value)
				VALUES ($1, $2)
				ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
			`, m.ID, *m.Value)
			if err != nil {
				return err
			}
		case "counter":
			if m.Delta == nil {
				return fmt.Errorf("missing counter delta for metric %s", m.ID)
			}
			_, err := tx.Exec(`
				INSERT INTO counters (name, delta)
				VALUES ($1, $2)
				ON CONFLICT (name) DO UPDATE SET delta = counters.delta + EXCLUDED.delta
			`, m.ID, *m.Delta)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown metric type: %s", m.MType)
		}
	}

	return tx.Commit()
}

func (r *SQLRepo) Ping() error {
	return r.db.Ping()
}
