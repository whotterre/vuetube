# VueTube
Backend for a YouTube-ish video streaming site, built off [this article](https://blog.kunalgoel.dev/designing-youtube-s-frontend-system-streams-feeds-and-scale?utm_source=hashnode&utm_medium=feed) about how YouTube's frontend is designed.

I started this mostly to figure out what DASH actually was — I'd seen `.m3u8` files and `.mpd` files floating around and wanted to know what was going on. Also an excuse to finally touch some AWS services I'd been avoiding.

[DB diagram](./docs/db_diagram.png)

## Stack
- Go + Gin
- PostgreSQL (user/video metadata)
- sqlc (generates the DB layer from raw SQL — really nice)
- AWS S3 (video + thumbnail storage)
- ffmpeg / ffprobe (metadata extraction, thumbnail generation)

## Getting started

You'll need:
- Go 1.22+
- PostgreSQL running somewhere
- ffmpeg in your PATH (`ffprobe` needs to be findable)
- sqlc if you're touching the DB queries (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)

**1. Set up your `.env` at `src/.env`:**

```
DATABASE_URL=postgres://postgres:password@localhost:5432/vuetube
JWT_SECRET=something-long-and-random
PORT=:8000
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret
AWS_REGION=us-east-1
BUCKET_NAME=your-bucket
```

For the AWS keys — go to IAM, create a user, attach `AmazonS3FullAccess`, generate access keys under Security Credentials.

**2. Apply the schema:**

```bash
psql $DATABASE_URL -f sql/schema.sql
```

**3. Run it:**

```bash
go run ./src/cmd
```

Or build:
```bash
go build -o src/cmd/cmd.exe ./src/cmd && ./src/cmd/cmd.exe
```

If you change any SQL queries or schema, regenerate the DB code:
```bash
sqlc generate
```

## Env vars

| Variable               | What it's for                     | Required?            |
|------------------------|-----------------------------------|----------------------|
| `DATABASE_URL`         | Postgres connection string        | yes                  |
| `JWT_SECRET`           | Signs the JWTs                    | yes                  |
| `PORT`                 | Which port to listen on           | no (default `:8000`) |
| `AWS_ACCESS_KEY_ID`    | AWS creds                         | for uploads          |
| `AWS_SECRET_ACCESS_KEY`| AWS creds                         | for uploads          |
| `AWS_REGION`           | Region your S3 bucket is in       | for uploads          |
| `BUCKET_NAME`          | S3 bucket for videos + thumbnails | for uploads          |

## Endpoints

### No auth needed

**GET /health** — just returns `{ "message": "Hello" }`. Useful for checking the server's up.

**POST /auth/signup**
```json
{
  "first_name": "Alice",
  "last_name": "Smith",
  "email": "alice@example.com",
  "password": "plaintext-password"
}
```
Returns a JWT on success (201).

**POST /auth/login**
```json
{
  "email": "alice@example.com",
  "password": "plaintext-password"
}
```
Returns a JWT on success (200).

### Auth required (`Authorization: Bearer <token>`)

**POST /videos/upload** — `multipart/form-data`

Fields:
- `video_file` — the actual video (up to 500 MB)
- `title` — required

What it does behind the scenes: runs ffprobe to get duration + resolution, uploads the video to S3, extracts a thumbnail from the second frame with ffmpeg, uploads that too, then saves everything to the DB.

Response (200):
```json
{
  "ID": "a533793a-acaf-465a-90e6-fc1700ac743d",
  "Name": "my-video.mp4",
  "S3Url": "https://<bucket>.s3.<region>.amazonaws.com/<hash>",
  "ThumbnailUrl": "https://<bucket>.s3.<region>.amazonaws.com/thumb-<hash>",
  "Duration": 16,
  "Resolution": "1080p",
  "Size": 6374470,
  "Progress": 0,
  "ViewCount": 0,
  "Owner": "<user-uuid>",
  "UploadedAt": "2026-05-28T21:19:00Z",
  "UpdatedAt": "2026-05-28T21:19:00Z"
}
```

![Upload response](./docs/slow_upload_ep.png)

## How the upload works

```
POST /videos/upload
  → RequireAuth middleware     validates JWT, sticks claims in Gin context
  → VideoHandler               checks file size, required fields
  → VideoService
      ├── ExtractMetadata      ffprobe on a temp file → duration + resolution
      ├── S3 upload            video goes up
      ├── ExtractThumbnail     ffmpeg grabs second frame → temp JPEG
      ├── S3 upload            thumbnail goes up (non-fatal if this fails)
      └── CreateVideo          record saved to postgres
```

Layer rules I tried to stick to:
- Handlers deal with HTTP only — no business logic
- Services do the work — they never touch `ctx.JSON` or write responses
- Repositories are just DB access via sqlc
- Middleware handles cross-cutting stuff (auth puts JWT claims in context under `"claims"`)

S3 keys are `md5(filename)` for videos and `thumb-md5(filename)` for thumbnails — deterministic, so re-uploading the same filename just overwrites.

## Known issues

### It's slow

Uploading a 6.7 MB file took ~25 seconds. The whole pipeline is synchronous — every step blocks until the previous one finishes:

```
receive → temp file → ffprobe × 2 → S3 upload → temp file → ffmpeg → S3 upload → postgres → respond
```

The obvious fix is to accept the file, immediately return a job ID, and do all the heavy lifting in the background. The `progress` field on the video record exists for exactly this — the plan is to use [hibiken/asynq](https://github.com/hibiken/asynq) for the job queue so the client can poll status.

On switching to an async workflow, I was able to realize a response time ~200x faster than the previous.
![Upload response after asynq](./docs/upload_after_asynq.png)

## Misc

- No automated migrations — just run `sql/schema.sql` manually against your DB
- The `sqlc` generated code is at `src/internal/db/sqlc` — don't edit it directly, change the SQL files and re-run `sqlc generate`
- JWT auth is required for video upload — the owner UUID is pulled directly from the token claims, so there's no way to upload anonymously

## Recommendation Feed

**GET /videos/feed/:id?limit=20** — returns a ranked list of videos recommended based on a seed video the user just watched or liked.

Rate limited to 10 req/s. `limit` defaults to 20, capped at 50.

Response (200):
```json
{
  "message": "recommendation feed fetched successfully",
  "recommendations": [
    {
      "ID": "...",
      "Name": "some-video.mp4",
      "ViewCount": 1042,
      ...
    }
  ]
}
```

### How it works

Videos are tagged via the `video_category` table — a join table between `videos` and a `category_tag` string. One video can have multiple tags.

The feed is built from two pools:

```
Seed video
  ├── Pool A — same tag set, ordered by views + recency     (~60% of results)
  └── Pool B — different tags / no overlap, serendipitous   (~40% of results)
```

This is implemented as a single CTE query with a `UNION ALL` fallback:

```sql
WITH cross_pool AS (
  -- Pool A: different-category videos, ordered by engagement
  SELECT v.* FROM videos v
  WHERE v.id != $1
    AND v.progress > 0
    AND v.id NOT IN (
      SELECT video_id FROM video_category
      WHERE category_tag = ANY(
        SELECT category_tag FROM video_category WHERE video_id = $1
      )
    )
  ORDER BY v.view_count DESC, v.uploaded_at DESC
  LIMIT $2
)
SELECT * FROM cross_pool
UNION ALL
(
  -- Pool B (fallback): fires only when cross_pool is empty — random for serendipity
  SELECT v.* FROM videos v
  WHERE v.id != $1
    AND NOT EXISTS (SELECT 1 FROM cross_pool)
  ORDER BY RANDOM()
  LIMIT $2
)
LIMIT $2;
```

The `NOT EXISTS (SELECT 1 FROM cross_pool)` guard means the fallback branch only activates when the primary pool returns nothing. When the fallback fires, results are randomised rather than popularity-sorted — intentional, to surface unexpected content.

### Cold start

New users launching the app for the first time have no seed video. A separate endpoint handles this — it returns the most popular and freshest content globally, with no personalisation.

**GET /videos/feed?limit=20**

No `:id` param. Rate limited to 10 req/s. `limit` defaults to 20, capped at 50.

Response (200):
```json
{
  "message": "generic feed fetched successfully",
  "feed": [
    {
      "ID": "...",
      "Name": "some-video.mp4",
      "ViewCount": 1042,
      ...
    }
  ]
}
```

Query used:
```sql
SELECT * FROM videos
WHERE progress > 0
ORDER BY view_count DESC, uploaded_at DESC
LIMIT $1;
```

Once the user watches or likes a video, switch to `GET /videos/feed/:id` to get personalised recommendations seeded from that video.


### Tagging videos

Tags aren't set on upload yet — insert them manually or wire them into the upload flow:

```sql
INSERT INTO video_category (video_id, category_tag)
VALUES ('<video-uuid>', 'gaming');
```

### Performance

Benchmarked with `EXPLAIN (ANALYZE, BUFFERS)` on a dev dataset (14 rows):

| Metric         | Value                                    |
|----------------|------------------------------------------|
| Execution time | 0.185 ms                                 |
| Planning time  | 0.657 ms                                 |
| Buffer hits    | 12 (all from cache, zero disk I/O)       |
| Tag lookup     | Index Only Scan on `video_category_pkey` |
| Fallback path  | Correctly skipped (`never executed`)     |

The `Seq Scan on videos` is expected and optimal at small scale — the planner switches to an index scan once the table grows. When you have thousands of videos, add:

```sql
-- Covers the ORDER BY in the primary pool
CREATE INDEX idx_videos_feed ON videos (view_count DESC, uploaded_at DESC)
WHERE progress > 0;

-- Covers the NOT IN tag lookup
CREATE INDEX idx_video_category_tag ON video_category (category_tag, video_id);
```

Re-run `EXPLAIN ANALYZE` after adding these — `Seq Scan` should become `Index Scan` and cost should drop significantly.