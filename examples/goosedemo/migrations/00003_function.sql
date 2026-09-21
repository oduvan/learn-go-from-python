-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION goose_touch() RETURNS trigger AS $$
BEGIN
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION goose_touch();
-- +goose StatementEnd
