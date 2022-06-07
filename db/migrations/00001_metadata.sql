-- +goose Up
-- +goose StatementBegin

CREATE TABLE metadata (
  id UUID PRIMARY KEY NOT NULL,
  metadata json NOT NULL default '{}',
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

COMMENT ON COLUMN metadata.id is 'The instance ID';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE metadata;

-- +goose StatementEnd
