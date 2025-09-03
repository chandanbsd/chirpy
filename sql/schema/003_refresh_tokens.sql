-- +goose Up
create table refresh_tokens (
    token text not null,
    created_at timestamp not null,
    updated_at timestamp not null,
    user_id UUID not null references users(id) on delete cascade,
    expires_at timestamp not null,
    revoked_at timestamp null
);

-- +goose Down
drop table refresh_tokens;
