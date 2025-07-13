-- +goose Up
-- +goose StatementBegin
CREATE TABLE Payment(
    requested_at   TEXT NOT NULL,
    correlation_id TEXT NOT NULL,
    processor      TEXT NOT NULL,
    amount         REAL NOT NULL,

    PRIMARY KEY(requested_at, correlation_id, processor)

);
CREATE UNIQUE INDEX cid_idx ON Payment(correlation_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX cid_idx;
DROP TABLE Payment;
-- +goose StatementEnd
