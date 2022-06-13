-- +goose Up
-- +goose StatementBegin

CREATE TABLE instance_userdata (
  id UUID PRIMARY KEY NOT NULL,
  userdata bytes,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

COMMENT ON COLUMN instance_userdata.id is 'The instance ID';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE instance_userdata;

-- +goose StatementEnd
