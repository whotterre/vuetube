-- name: CreateVideo :one
INSERT INTO "videos" (name, size, owner)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateVideoAfterProcessing :one
UPDATE "videos"
SET
  s3_url       = $2,
  thumbnail_url = $3,
  duration     = $4,
  resolution   = $5,
  progress     = 100,
  updated_at   = now()
WHERE id = $1
RETURNING *;


-- name: DeleteVideoLike :exec
DELETE FROM video_likes
WHERE video_id = $1 AND user_id = $2;

-- name: InsertVideoLike :exec
INSERT INTO video_likes (video_id, user_id)
VALUES ($1, $2);

-- name: FindVideoByVideoId :one
SELECT * FROM videos WHERE id = $1;

-- name: CheckUserLikedVideo :one
SELECT * FROM video_likes WHERE video_id = $1 AND user_id = $2;

-- name: CountVideoLikes :one
SELECT COUNT(*) FROM video_likes WHERE video_id = $1;

