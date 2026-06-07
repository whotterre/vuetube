CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    "email" text NOT NULL UNIQUE,
    "first_name" text NOT NULL,
    "last_name" text NOT NULL,
    "country" text NOT NULL DEFAULT '',
    "password" text NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now(),
    "deleted_at" timestamptz
);

CREATE TABLE IF NOT EXISTS "videos" (
  "id" uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  "name" varchar NOT NULL,
  "s3_url" varchar NOT NULL DEFAULT '',
  "thumbnail_url" varchar NOT NULL DEFAULT '',
  "duration" int NOT NULL DEFAULT 0,
  "resolution" varchar NOT NULL DEFAULT '',
  "size" int NOT NULL,
  "progress" int NOT NULL DEFAULT 0,
  "view_count" int NOT NULL DEFAULT 0,
  "owner" uuid REFERENCES users(id),
  "uploaded_at" timestamp NOT NULL DEFAULT (now()),
  "updated_at" timestamp NOT NULL DEFAULT (now())
);

CREATE TABLE IF NOT EXISTS "video_likes" (
  video_id   UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
  created_at timestamptz DEFAULT now(),
  PRIMARY KEY (video_id, user_id)
);

CREATE TABLE IF NOT EXISTS "video_category" (
  video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
  category_tag varchar(255) NOT NULL,
  PRIMARY KEY (video_id, category_tag)
);

CREATE INDEX IF NOT EXISTS idx_videos_feed ON videos (progress, view_count DESC, uploaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_videos_owner ON videos (owner);
CREATE INDEX IF NOT EXISTS idx_video_likes_user ON video_likes (user_id);
CREATE INDEX IF NOT EXISTS idx_video_category_tag ON video_category (category_tag);