package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type ConfigStore struct {
	DB *sql.DB
}

// ActiveConfig is the full algorithm configuration as seen by the dashboard.
// "name" is the internal strategy class (e.g. EngagementStrategy) used by the
// StrategyFactory, while "strategy_name" is the human-facing label the UI uses
// (engagement / chronological / diversity).
type ActiveConfig struct {
	Name         string  `json:"name"`
	StrategyName string  `json:"strategy_name"`
	Weight       float64 `json:"weight"`
	UpdatedAt    string  `json:"updated_at"`
}

// strategyClassFor maps a UI label to the internal strategy class name.
func strategyClassFor(strategyName string) string {
	switch strategyName {
	case "chronological":
		return "ChronologicalStrategy"
	case "engagement", "diversity":
		return "EngagementStrategy"
	default:
		return "EngagementStrategy"
	}
}

// GetActiveStrategy returns the internal strategy class name (used by ranking).
func (s *ConfigStore) GetActiveStrategy() string {
	cfg := s.GetActiveConfig()
	if cfg.Name == "" {
		return "EngagementStrategy"
	}
	return cfg.Name
}

// GetActiveConfig reads the full active configuration, defaulting gracefully.
func (s *ConfigStore) GetActiveConfig() ActiveConfig {
	def := ActiveConfig{Name: "EngagementStrategy", StrategyName: "engagement", Weight: 0.85}
	var raw []byte
	err := s.DB.QueryRow("SELECT value FROM configs WHERE key='active_strategy' LIMIT 1").Scan(&raw)
	if err != nil {
		log.Println("could not read config, defaulting to EngagementStrategy:", err)
		return def
	}
	var cfg ActiveConfig
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

// UpsertActiveConfig persists a new active configuration from a UI label.
func (s *ConfigStore) UpsertActiveConfig(strategyName string, weight float64) (ActiveConfig, error) {
	cfg := ActiveConfig{
		Name:         strategyClassFor(strategyName),
		StrategyName: strategyName,
		Weight:       weight,
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	raw, _ := json.Marshal(cfg)
	_, err := s.DB.Exec(
		"INSERT INTO configs (key, value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=$2",
		"active_strategy", raw,
	)
	return cfg, err
}

// ConfigChange is one persisted deployment-log entry.
type ConfigChange struct {
	StrategyName string  `json:"strategy_name"`
	Weight       float64 `json:"weight"`
	UpdatedAt    string  `json:"updated_at"`
}

// AddHistory appends a deployment-log entry so the dashboard's "Deployment Logs"
// survive page navigation and service restarts.
func (s *ConfigStore) AddHistory(strategyName string, weight float64) error {
	_, err := s.DB.Exec(
		"INSERT INTO config_history (strategy_name, weight) VALUES ($1,$2)",
		strategyName, weight,
	)
	return err
}

// GetHistory returns the most recent deployment-log entries, newest first.
func (s *ConfigStore) GetHistory(limit int) ([]ConfigChange, error) {
	rows, err := s.DB.Query(
		"SELECT strategy_name, weight, created_at FROM config_history ORDER BY created_at DESC LIMIT $1",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ConfigChange{}
	for rows.Next() {
		var c ConfigChange
		var ts time.Time
		if err := rows.Scan(&c.StrategyName, &c.Weight, &ts); err != nil {
			continue
		}
		c.UpdatedAt = ts.UTC().Format(time.RFC3339)
		out = append(out, c)
	}
	return out, nil
}
