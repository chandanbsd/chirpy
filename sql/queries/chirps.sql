-- name: CreateChirp :one
INSERT INTO chirp (id, created_at, updated_at, body, user_id)
VALUES (
       gen_random_uuid(),
        NOW(),
        NOW(),
	$1,
	$2
       )
RETURNING *;

-- name: DeleteChirps :exec
DELETE FROM chirp;

-- name: GetChirps :many
SELECT id,
       created_at as "created_at",
       updated_at as "updated_at",
       body as "body",
       user_id as "user_id"
FROM chirp
ORDER BY created_at asc;

-- name: GetChirp :one
SELECT id,
       created_at,
       updated_at,
       body,
       user_id
FROM chirp
WHERE id = $1;

-- name: DeleteChirp :exec
DELETE
from chirp
where id = $1;
