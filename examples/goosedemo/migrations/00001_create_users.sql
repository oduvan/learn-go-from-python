-- +goose Up
CREATE TABLE goose_users (
    id bigserial PRIMARY KEY,
    email text NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE goose_users;
