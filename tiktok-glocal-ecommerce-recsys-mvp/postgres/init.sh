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
CREATE TABLE IF NOT EXISTS profiles (
  user_id text primary key,
  tags jsonb
);
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
INSERT INTO videos (video_id, author, category, title) VALUES
  ('v1','author1','fashion','Red Dress'),
  ('v2','author2','electronics','Wireless Earbuds'),
  ('v3','author3','home','Ceramic Vase')
ON CONFLICT DO NOTHING;
EOSQL

echo "Creating tables in rec_db..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname=rec_db <<-EOSQL
CREATE TABLE IF NOT EXISTS configs (
  id serial primary key,
  key text unique,
  value jsonb
);
INSERT INTO configs (key, value) VALUES ('active_strategy', jsonb_build_object('name','EngagementStrategy')) ON CONFLICT (key) DO NOTHING;
EOSQL

echo "Init complete."
