-- name: CreateUser :one
INSERT INTO users (email, password)
VALUES ($1, $2)
RETURNING id, email, password, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password, created_at, updated_at
FROM users
WHERE email = $1
  AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, email, password, created_at, updated_at
FROM users
WHERE id = $1
  AND deleted_at IS NULL;