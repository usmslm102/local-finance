-- +goose Up
CREATE TABLE savings_goals (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    target_amount REAL NOT NULL CHECK(target_amount > 0),
    saved_amount REAL NOT NULL DEFAULT 0 CHECK(saved_amount >= 0),
    monthly_contribution REAL NOT NULL DEFAULT 0 CHECK(monthly_contribution >= 0),
    target_date TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- +goose Down
DROP TABLE savings_goals;
