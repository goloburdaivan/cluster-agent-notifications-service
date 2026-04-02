-- name: ListChannels :many
SELECT id, type, credentials, name, enabled
FROM channels;

-- name: GetActiveChannels :many
SELECT id, type, credentials, name
FROM channels
WHERE enabled = true;

-- name: GetChannel :one
SELECT id, type, credentials, name, enabled
FROM channels
WHERE id = $1;

-- name: CreateChannel :one
INSERT INTO channels (name, credentials, type, enabled)
VALUES ($1, $2, $3, $4)
RETURNING id, name, credentials, type, enabled;

-- name: UpdateChannel :one
UPDATE channels
SET name        = $2,
    credentials = $3,
    type        = $4,
    enabled     = $5
WHERE id = $1
RETURNING id, name, credentials, type, enabled;

-- name: DeleteChannel :execrows
DELETE
FROM channels
WHERE id = $1;