-- +goose Up
-- +goose StatementBegin

CREATE INDEX index_instance_ip_addresses_instance_id ON instance_ip_addresses (instance_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX index_instance_ip_addresses_instance_id;

-- +goose StatementEnd
