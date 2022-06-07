-- +goose Up
-- +goose StatementBegin

CREATE TABLE userdata (
  id UUID PRIMARY KEY NOT NULL,
  userdata bytes,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

COMMENT ON COLUMN userdata.id is 'The instance ID';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE userdata;

-- +goose StatementEnd
