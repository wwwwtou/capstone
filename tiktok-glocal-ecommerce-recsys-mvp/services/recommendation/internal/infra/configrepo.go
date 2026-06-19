package infra

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	_ "github.com/lib/pq"

	"recommendation/internal/domain"
)

// PostgresConfigRepository implements domain.ConfigRepository against rec_db.
type PostgresConfigRepository struct {
	db *sql.DB
}

func NewPostgresConfigRepository(db *sql.DB) *PostgresConfigRepository {
	return &PostgresConfigRepository{db: db}
}

// ActiveStrategy returns the internal strategy class name used by ranking.
func (r *PostgresConfigRepository) ActiveStrategy() string {
	cfg := r.ActiveConfig()
	if cfg.Name == "" {
		return "EngagementStrategy"
	}
	return cfg.Name
}

// ActiveConfig reads the full active configuration, defaulting gracefully.
func (r *PostgresConfigRepository) ActiveConfig() domain.ActiveConfig {
	def := domain.ActiveConfig{Name: "EngagementStrategy", StrategyName: "engagement", Weight: 0.85}
	var raw []byte
	err := r.db.QueryRow("SELECT value FROM configs WHERE key='active_strategy' LIMIT 1").Scan(&raw)
	if err != nil {
		log.Println("could not read config, defaulting to EngagementStrategy:", err)
		return def
	}
	var cfg domain.ActiveConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return def
	}
	if cfg.Name == "" {
		cfg.Name = "EngagementStrategy"
	}
	if cfg.StrategyName == "" {
		cfg.StrategyName = "engagement"
	}
	return cfg
}

// UpsertConfig persists a new active configuration from a UI label.
func (r *PostgresConfigRepository) UpsertConfig(strategyName string, weight float64) (domain.ActiveConfig, error) {
	cfg := domain.ActiveConfig{
		Name:         domain.StrategyClassFor(strategyName),
		StrategyName: strategyName,
		Weight:       weight,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	raw, _ := json.Marshal(cfg)
	_, err := r.db.Exec(
		"INSERT INTO configs (key, value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=$2",
		"active_strategy", raw,
	)
	return cfg, err
}

// AddHistory appends a deployment-log entry.
func (r *PostgresConfigRepository) AddHistory(strategyName string, weight float64) error {
	_, err := r.db.Exec(
		"INSERT INTO config_history (strategy_name, weight) VALUES ($1,$2)",
		strategyName, weight,
	)
	return err
}

// History returns the most recent deployment-log entries, newest first.
func (r *PostgresConfigRepository) History(limit int) ([]domain.ConfigChange, error) {
	rows, err := r.db.Query(
		"SELECT strategy_name, weight, created_at FROM config_history ORDER BY created_at DESC LIMIT $1",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ConfigChange{}
	for rows.Next() {
		var c domain.ConfigChange
		var ts time.Time
		if err := rows.Scan(&c.StrategyName, &c.Weight, &ts); err != nil {
			continue
		}
		c.UpdatedAt = ts.UTC().Format(time.RFC3339)
		out = append(out, c)
	}
	return out, nil
}
