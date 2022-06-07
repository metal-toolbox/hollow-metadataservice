-- +goose Up
-- +goose StatementBegin

CREATE TABLE instance_ip_addresses (
  id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  instance_id UUID NOT NULL,
  address INET NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
)

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE instance_ip_addresses;

-- +goose StatementEnd
