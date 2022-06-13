-- +goose Up
-- +goose StatementBegin

CREATE TABLE instance_metadata (
  id UUID PRIMARY KEY NOT NULL,
  metadata json NOT NULL default '{}',
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

COMMENT ON COLUMN instance_metadata.id is 'The instance ID';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE instance_metadata;

-- +goose StatementEnd
