package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/ternaryinvalid/balancer/internal/config"
)

type DB struct {
	conn *sql.DB
}

func Connect(dbConfig config.DB) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbConfig.Host, dbConfig.Port,
		dbConfig.User, dbConfig.Password,
		dbConfig.DBName,
	)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("не получилось подключиться к БД %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("БД недоступна: %w", err)
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) CreateRateLimitTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS rate_limit_rules (
		client_id   TEXT PRIMARY KEY,
		capacity    INTEGER NOT NULL,
		refill_rate INTEGER NOT NULL
	)`

	_, err := db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create rate limit table: %w", err)
	}
	return nil
}

func (db *DB) GetRateLimitRule(clientID string) (capacity, refillRate int, err error) {
	query := `SELECT capacity, refill_rate FROM rate_limit_rules WHERE client_id = $1`

	err = db.conn.QueryRow(query, clientID).Scan(&capacity, &refillRate)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, fmt.Errorf("правила для клиента не найдены")
		}
		return 0, 0, fmt.Errorf("ошибка при получении правил: %w", err)
	}

	return capacity, refillRate, nil
}

func (db *DB) SetRateLimitRule(clientID string, capacity, refillRate int) error {
	query := `
    INSERT INTO rate_limit_rules (client_id, capacity, refill_rate)
    VALUES ($1, $2, $3)
    ON CONFLICT (client_id) DO UPDATE 
    SET capacity = EXCLUDED.capacity, refill_rate = EXCLUDED.refill_rate`

	_, err := db.conn.Exec(query, clientID, capacity, refillRate)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении правил: %w", err)
	}

	return nil
}
