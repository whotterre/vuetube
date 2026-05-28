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

| Variable | What it's for | Required? |
|---|---|---|
| `DATABASE_URL` | Postgres connection string | yes |
| `JWT_SECRET` | Signs the JWTs | yes |
| `PORT` | Which port to listen on | no (default `:8000`) |
| `AWS_ACCESS_KEY_ID` | AWS creds | for uploads |
| `AWS_SECRET_ACCESS_KEY` | AWS creds | for uploads |
| `AWS_REGION` | Region your S3 bucket is in | for uploads |
| `BUCKET_NAME` | S3 bucket for videos + thumbnails | for uploads |

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

## Misc

- No automated migrations — just run `sql/schema.sql` manually against your DB
- The `sqlc` generated code is at `src/internal/db/sqlc` — don't edit it directly, change the SQL files and re-run `sqlc generate`
- JWT auth is required for video upload — the owner UUID is pulled directly from the token claims, so there's no way to upload anonymously