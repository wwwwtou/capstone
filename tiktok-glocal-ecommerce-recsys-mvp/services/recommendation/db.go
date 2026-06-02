package main

import (
	"database/sql"
	"encoding/json"
	"log"

	_ "github.com/lib/pq"
)

type ConfigStore struct {
	DB *sql.DB
}

func (s *ConfigStore) GetActiveStrategy() string {
	var raw []byte
	err := s.DB.QueryRow("SELECT value FROM configs WHERE key='active_strategy' LIMIT 1").Scan(&raw)
	if err != nil {
		log.Println("could not read config, defaulting to EngagementStrategy:", err)
		return "EngagementStrategy"
	}
	var v map[string]interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "EngagementStrategy"
	}
	if name, ok := v["name"].(string); ok {
		return name
	}
	return "EngagementStrategy"
}
