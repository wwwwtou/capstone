-- TikTok Glocal E-commerce Initial Schema

-- 1. Algorithm Configuration Table
CREATE TABLE algorithm_configs (
    id SERIAL PRIMARY KEY,
    strategy_name VARCHAR(50) NOT NULL, -- 'engagement' or 'chronological'
    weight DECIMAL(3, 2) DEFAULT 0.5,
    is_active BOOLEAN DEFAULT FALSE,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 2. Indexing is critical for MTech defense (Architecture Decision)
CREATE INDEX idx_strategy_active ON algorithm_configs(is_active) WHERE is_active = TRUE;

-- 3. Video Metadata Table
CREATE TABLE video_metadata (
    id VARCHAR(50) PRIMARY KEY,
    title TEXT NOT NULL,
    author_id VARCHAR(50) NOT NULL,
    category VARCHAR(30) NOT NULL,
    tags JSONB, -- Storing flexible metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_video_category ON video_metadata(category);

-- 4. Initial Seed Data
INSERT INTO algorithm_configs (strategy_name, weight, is_active) VALUES 
('engagement', 0.85, TRUE),
('chronological', 0.50, FALSE);

INSERT INTO video_metadata (id, title, author_id, category, tags) VALUES 
('v_01', 'Best Mechanical Keyboards 2026', 'user_a', 'tech', '{"brand": "Keychron", "type": "mechanical"}'),
('v_02', 'Street Food in Jakarta', 'user_b', 'food', '{"location": "Indonesia", "halal": true}'),
('v_03', 'Home Workout No Equipment', 'user_c', 'fitness', '{"level": "beginner", "duration": "10m"}');
