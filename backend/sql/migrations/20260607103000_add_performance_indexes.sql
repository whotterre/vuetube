CREATE INDEX IF NOT EXISTS idx_videos_feed ON videos (progress, view_count DESC, uploaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_videos_owner ON videos (owner);
CREATE INDEX IF NOT EXISTS idx_video_likes_user ON video_likes (user_id);
CREATE INDEX IF NOT EXISTS idx_video_category_tag ON video_category (category_tag);
