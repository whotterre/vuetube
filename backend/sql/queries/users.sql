-- name: CreateUser :one
INSERT INTO users (first_name, last_name, email, password)
VALUES ($1, $2, $3, $4)
RETURNING id, email, first_name, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, first_name, last_name, email, password, created_at, updated_at
FROM users
WHERE email = $1
  AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, email, password, created_at, updated_at
FROM users
WHERE id = $1
  AND deleted_at IS NULL;