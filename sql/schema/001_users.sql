-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP  not null,
    updated_at TIMESTAMP not null,
    email TEXT not null,
    hashed_password text not null default 'unset'
);

-- +goose Down
DROP TABLE users;
