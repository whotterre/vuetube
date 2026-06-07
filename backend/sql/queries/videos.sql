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

-- name: GetCrossPoolRecommendations :many
WITH cross_pool AS (
  SELECT v.* FROM videos v
  WHERE v.id != $1
    AND v.progress > 0
    AND v.id NOT IN (
      SELECT video_id FROM video_category
      WHERE category_tag = ANY(SELECT category_tag FROM video_category WHERE video_id = $1)
    )
  ORDER BY v.view_count DESC, v.uploaded_at DESC
  LIMIT $2
)
SELECT * FROM cross_pool
UNION ALL
(
  SELECT v.* FROM videos v
  WHERE v.id != $1
    AND NOT EXISTS (SELECT 1 FROM cross_pool)
  ORDER BY RANDOM()
  LIMIT $2
)
LIMIT $2;


-- name: IncrementViewCount :exec
UPDATE "videos" SET view_count = view_count + 1 WHERE id = $1;

-- name: GetGenericFeed :many
SELECT * FROM videos
WHERE progress > 0
ORDER BY view_count DESC, uploaded_at DESC
LIMIT $1;


