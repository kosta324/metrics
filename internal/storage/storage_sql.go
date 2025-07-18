package storage

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

type SQLRepo struct {
	db *sql.DB
}

func NewSQLStorage(dsn string) (*SQLRepo, error) {
	db, err := sql.Open("postgres", dsn)
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
	switch metricType {
	case "gauge":
		_, err := r.db.Exec(`
            INSERT INTO gauges (name, value)
            VALUES ($1, $2)
            ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
        `, name, value)
		return err
	case "counter":
		_, err := r.db.Exec(`
            INSERT INTO counters (name, delta)
            VALUES ($1, $2)
            ON CONFLICT (name) DO UPDATE SET delta = counters.delta + EXCLUDED.delta
        `, name, value)
		return err
	default:
		return fmt.Errorf("unsupported metric type: %s", metricType)
	}
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
