-- +goose Up
CREATE TABLE mcp_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled BOOLEAN NOT NULL DEFAULT 0,
    port INTEGER NOT NULL DEFAULT 8081 CHECK (port BETWEEN 1024 AND 65535),
    token_hash TEXT NOT NULL DEFAULT ''
);
INSERT INTO mcp_settings (id) VALUES (1);

-- +goose Down
DROP TABLE mcp_settings;
