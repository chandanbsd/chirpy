-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
       gen_random_uuid(),
        NOW(),
        NOW(),
        $1,
        $2
       )
RETURNING *;

-- name: DeleteUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT id,
       created_at,
       updated_at,
       email,
       hashed_password,
       is_chirpy_red
FROM users
WHERE email = $1;

-- name: DoesUserExist :one
SELECT id
from users
WHERE id = $1;

-- name: UpdateUserCredential :exec
UPDATE users
SET email = $2,
    hashed_password = $3
WHERE id = $1;

-- name: GetUserById :one
SELECT id,
    created_at,
    updated_at,
    email,
    is_chirpy_red
from users
where id = $1;

-- name: UpgradeUserToChirpyRed :exec
UPDATE users
SET is_chirpy_red = true
WHERE id = $1;
