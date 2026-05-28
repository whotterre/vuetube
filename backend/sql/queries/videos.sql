-- name: CreateVideo :one
INSERT INTO "videos" 
(name, s3_url, thumbnail_url, duration, resolution, size, progress, view_count, owner) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

