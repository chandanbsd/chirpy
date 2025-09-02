-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    token,
    created_at,
    updated_at,
    user_id,
    expires_at,
    revoked_at
)
VALUES(
    $1,
    NOW(),
    NOW(),
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetRefreshToken :one
select
    token,
    created_at,
    updated_at,
    user_id,
    expires_at,
    revoked_at
from refresh_tokens
where token = $1;

-- name: RevokeToken :exec
update refresh_tokens
set revoked_at = NOW()
where token = $1;
