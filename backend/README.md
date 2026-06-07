# VueTube
Backend for a YouTube-ish video streaming site, built off [this article](https://blog.kunalgoel.dev/designing-youtube-s-frontend-system-streams-feeds-and-scale?utm_source=hashnode&utm_medium=feed) about how YouTube's frontend is designed.

I started this mostly to figure out what DASH actually was — I'd seen `.m3u8` files and `.mpd` files floating around and wanted to know what was going on. Also an excuse to finally touch some AWS services I'd been avoiding.

[DB diagram](./docs/db_diagram.png)

## Stack
- Go + Gin
- PostgreSQL (user/video metadata)
- sqlc (generates the DB layer from raw SQL)
- Redis + Asynq (background video processing jobs)
- AWS S3 (DASH manifests, media chunks, and thumbnails)
- ffmpeg / ffprobe (metadata extraction, thumbnail generation, DASH packaging)

## Getting started

You'll need:
- Go 1.22+
- PostgreSQL running somewhere
- Redis running somewhere (Asynq uses this for the upload queue)
- ffmpeg in your PATH (`ffprobe` needs to be findable)
- sqlc if you're touching the DB queries (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)

**1. Set up your `.env` at `src/.env`:**

```
DATABASE_URL=postgres://postgres:password@localhost:5432/vuetube
JWT_SECRET=something-long-and-random
PORT=:8000
REDIS_ADDR=localhost:6379
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

- `DATABASE_URL` - required. Postgres connection string.
- `JWT_SECRET` - required. Used to sign JWTs.
- `PORT` - optional, defaults to `:8000`. Controls which port the server listens on.
- `REDIS_ADDR` - required. Redis address for Asynq jobs.
- `AWS_ACCESS_KEY_ID` - required for uploads. AWS access key.
- `AWS_SECRET_ACCESS_KEY` - required for uploads. AWS secret key.
- `AWS_REGION` - required for uploads. Region your S3 bucket is in.
- `BUCKET_NAME` - required for uploads. S3 bucket for DASH assets and thumbnails.

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

What it does behind the scenes: saves the upload to a temp file, creates a video row with `progress = 0`, enqueues an Asynq job, and returns immediately. The worker then runs ffprobe/ffmpeg, uploads the DASH assets and thumbnail to S3, and updates the DB row with the manifest URL.

Initial response (200):
```json
{
  "ID": "a533793a-acaf-465a-90e6-fc1700ac743d",
  "Name": "my-video.mp4",
  "S3Url": "",
  "ThumbnailUrl": "",
  "Duration": 0,
  "Resolution": "",
  "Size": 6374470,
  "Progress": 0,
  "ViewCount": 0,
  "Owner": "<user-uuid>",
  "UploadedAt": "2026-05-28T21:19:00Z",
  "UpdatedAt": "2026-05-28T21:19:00Z"
}
```

After the worker finishes, `S3Url` points at the DASH manifest:

```json
{
  "S3Url": "https://<bucket>.s3.<region>.amazonaws.com/videos/<video-id>/dash/index.mpd",
  "ThumbnailUrl": "https://<bucket>.s3.<region>.amazonaws.com/thumb-<video-id>",
  "Duration": 16,
  "Resolution": "1080p",
  "Progress": 100
}
```

![Upload response](./docs/slow_upload_ep.png)

**GET /videos/:id/dash/manifest** - returns a rewritten DASH manifest (`application/dash+xml`).

The manifest is fetched from S3, validated as MPD XML, and rewritten so DASH init/media URLs point back through this backend:

```text
/videos/<id>/dash/segment/init-stream0.m4s
/videos/<id>/dash/segment/chunk-stream0-00001.m4s
```

**GET /videos/:id/dash/segment/:filename** - streams one DASH asset from S3.

Accepted filenames include:

```text
init.mp4
init-stream0.mp4
init-stream0.m4s
chunk-00001.m4s
chunk-stream0-00001.m4s
```

## How the upload works

Current flow:

```text
POST /videos/upload
  -> RequireAuth middleware      validates JWT, stores claims in Gin context
  -> VideoHandler                checks file size and required fields
  -> VideoService
      -> Save temp file          uploaded video goes to local temp storage
      -> CreateVideo             record saved to Postgres with progress = 0
      -> Enqueue Asynq job       returns to the client quickly

Asynq worker
  -> ExtractMetadata             ffprobe reads duration + resolution
  -> ExtractThumbnail            ffmpeg grabs an early frame -> temp JPEG
  -> Upload thumbnail            S3 key: thumb-<video-id>
  -> Inspect audio streams       ffprobe checks whether audio exists
  -> Package DASH                ffmpeg writes index.mpd + init-* + chunk-*
  -> Upload DASH assets          S3 prefix: videos/<video-id>/dash/
  -> UpdateVideoAfterProcessing  stores manifest URL, thumbnail URL, metadata, progress = 100
```

S3 keys are grouped by video ID:

```text
thumb-<video-id>
videos/<video-id>/dash/index.mpd
videos/<video-id>/dash/init-stream0.m4s
videos/<video-id>/dash/chunk-stream0-00001.m4s
videos/<video-id>/dash/chunk-stream1-00001.m4s
```

The DB `s3_url` stores the full URL for `index.mpd`, not the original uploaded video. The player loads that manifest, then requests init/chunk files through the backend segment proxy.

Older synchronous flow, kept here for comparison:

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

The current S3 key layout is the video-ID grouped DASH layout shown above. Older notes about `md5(filename)` keys only applied before the async DASH worker was added.

## Known issues

### Video processing still needs production hardening

Uploads return quickly now, but transcoding is still handled by a local worker. For production, this should probably move to ECS/Fargate, EC2, or AWS MediaConvert rather than running long ffmpeg jobs inside the web process.

Current limitations:

- Only one video rendition is generated, so this is DASH chunking but not true adaptive bitrate yet.
- No CDN layer; the backend proxies private S3 objects directly.
- No detailed progress updates during transcoding; the record mostly moves from `0` to `100`.
- Retry/idempotency is basic. A failed worker can leave partial DASH assets in S3.

On switching to an async workflow, upload response time became much faster than the first synchronous version.
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

New users launching the app for the first time have no seed video. A separate endpoint should handle this:

**Planned: GET /videos/feed** — no `:id` param, returns a generic popular/fresh feed ordered by `view_count DESC, uploaded_at DESC`.

### Tagging videos

Tags aren't set on upload yet — insert them manually or wire them into the upload flow:

```sql
INSERT INTO video_category (video_id, category_tag)
VALUES ('<video-uuid>', 'gaming');
```

### Performance

Benchmarked with `EXPLAIN (ANALYZE, BUFFERS)` on a dev dataset (14 rows):

- Execution time: `0.185 ms`
- Planning time: `0.657 ms`
- Buffer hits: `12`, all from cache with zero disk I/O
- Tag lookup: index-only scan on `video_category_pkey`
- Fallback path: correctly skipped (`never executed`)

The `Seq Scan on videos` is expected and optimal at small scale — the planner switches to an index scan once the table grows. When you have thousands of videos, add:

```sql
-- Covers the ORDER BY in the primary pool
CREATE INDEX idx_videos_feed ON videos (view_count DESC, uploaded_at DESC)
WHERE progress > 0;

-- Covers the NOT IN tag lookup
CREATE INDEX idx_video_category_tag ON video_category (category_tag, video_id);
```

Re-run `EXPLAIN ANALYZE` after adding these — `Seq Scan` should become `Index Scan` and cost should drop significantly.
