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
