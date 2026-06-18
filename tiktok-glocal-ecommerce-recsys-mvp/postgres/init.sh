#!/bin/bash
set -e

echo "Creating logical databases..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
CREATE DATABASE user_db;
CREATE DATABASE content_db;
CREATE DATABASE rec_db;
EOSQL

echo "Creating tables in user_db..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname=user_db <<-EOSQL
CREATE TABLE IF NOT EXISTS interactions (
  id serial primary key,
  user_id text,
  event_type text,
  metadata jsonb,
  created_at timestamptz default now()
);
CREATE INDEX IF NOT EXISTS idx_interactions_user ON interactions(user_id);
CREATE TABLE IF NOT EXISTS profiles (
  user_id text primary key,
  tags jsonb
);

-- Seed interactions for demo users so each one gets a *different* profile,
-- which makes the recommendation ranking visibly differ per user_id.
INSERT INTO interactions (user_id, event_type, metadata) VALUES
  ('user_123','view', '{"category":"electronics"}'),
  ('user_123','like', '{"category":"electronics"}'),
  ('user_123','view', '{"category":"tech"}'),
  ('user_123','view', '{"category":"tech"}'),
  ('user_fashion','view', '{"category":"fashion"}'),
  ('user_fashion','like', '{"category":"fashion"}'),
  ('user_fashion','view', '{"category":"home"}'),
  ('user_foodie','view', '{"category":"food"}'),
  ('user_foodie','like', '{"category":"food"}'),
  ('user_foodie','view', '{"category":"travel"}');
EOSQL

echo "Creating tables in content_db..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname=content_db <<-EOSQL
CREATE TABLE IF NOT EXISTS videos (
  id serial primary key,
  video_id text unique,
  author text,
  category text,
  title text,
  created_at timestamptz default now()
);
CREATE INDEX IF NOT EXISTS idx_videos_category ON videos(category);
-- Staggered created_at so the ChronologicalStrategy produces a meaningful order.
INSERT INTO videos (video_id, author, category, title, created_at) VALUES
  ('v1','StyleHouse','fashion','Autumn Streetwear Lookbook',      now() - interval '1 hour'),
  ('v2','TechMaster','electronics','Wireless Earbuds Deep Dive',  now() - interval '2 hour'),
  ('v3','HomeNest','home','Minimalist Ceramic Vase',              now() - interval '3 hour'),
  ('v4','FoodieIntl','food','Jakarta Street Food Tour',           now() - interval '4 hour'),
  ('v5','GadgetGuru','tech','Top Tech Gadgets 2026',              now() - interval '5 hour'),
  ('v6','FitLife','fitness','10-Minute Home Workout',             now() - interval '6 hour'),
  ('v7','TechMaster','electronics','Mechanical Keyboard Review',  now() - interval '7 hour'),
  ('v8','Wanderer','travel','Hidden Beaches of Bali',             now() - interval '8 hour'),
  ('v9','StyleHouse','fashion','Capsule Wardrobe Basics',         now() - interval '9 hour'),
  ('v10','GadgetGuru','tech','AI Phones Compared',                now() - interval '10 hour')
ON CONFLICT DO NOTHING;
EOSQL

echo "Creating tables in rec_db..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname=rec_db <<-EOSQL
CREATE TABLE IF NOT EXISTS configs (
  id serial primary key,
  key text unique,
  value jsonb
);
INSERT INTO configs (key, value) VALUES ('active_strategy', jsonb_build_object('name','EngagementStrategy','strategy_name','engagement','weight',0.85,'updated_at','seed')) ON CONFLICT (key) DO NOTHING;
-- Deployment-log history so the dashboard's "Deployment Logs" persist across
-- page navigation and service restarts.
CREATE TABLE IF NOT EXISTS config_history (
  id serial primary key,
  strategy_name text not null,
  weight numeric not null,
  created_at timestamptz default now()
);
EOSQL

echo "Init complete."
