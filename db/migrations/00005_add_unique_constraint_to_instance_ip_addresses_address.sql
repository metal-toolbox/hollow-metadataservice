-- +goose Up
-- +goose StatementBegin

ALTER TABLE instance_ip_addresses ADD CONSTRAINT unique_address UNIQUE (address);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP CONSTRAINT unique_address ON instance_ip_addresses;

-- +goose StatementEnd
