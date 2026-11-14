-- +goose Up
CREATE TABLE sessions
(
    phone    VARCHAR(32) PRIMARY KEY,
    data bytea
);

-- +goose Down
DROP TABLE sessions;
