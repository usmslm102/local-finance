-- +goose Up
-- Development databases may already have this table without the corresponding
-- Goose record. Reuse it and preserve its access settings when replaying.
CREATE TABLE IF NOT EXISTS mcp_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled BOOLEAN NOT NULL DEFAULT 0,
    port INTEGER NOT NULL DEFAULT 8081 CHECK (port BETWEEN 1024 AND 65535),
    token_hash TEXT NOT NULL DEFAULT ''
);
INSERT INTO mcp_settings (id, enabled, port, token_hash)
VALUES (1, 0, 8081, '')
ON CONFLICT(id) DO NOTHING;

-- +goose Down
DROP TABLE mcp_settings;
