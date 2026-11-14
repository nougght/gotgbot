-- +goose Up
CREATE TABLE messages
(
    phone    VARCHAR(32) NOT NULL,
    peer_type INT NOT NULL CHECK (peer_type BETWEEN 0 AND 2),
    peer_id BIGINT NOT NULL,
    from_id BIGINT,
    msg_id BIGINT NOT NULL,

    out BOOLEAN,
    content TEXT,
    date TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,

    PRIMARY KEY(phone, peer_type, peer_id, msg_id)
);
-- +goose Down
DROP TABLE messages;
