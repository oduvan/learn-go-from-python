-- +goose Up
ALTER TABLE goose_users ADD COLUMN name text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE goose_users DROP COLUMN name;
