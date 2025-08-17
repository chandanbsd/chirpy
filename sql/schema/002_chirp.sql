-- +goose Up
CREATE TABLE chirp (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP  not null,
    updated_at TIMESTAMP not null,
    body TEXT not null,
    user_id UUID references users(id) on delete cascade
);

-- +goose Down
DROP TABLE chirp;
