# VueTube
A video streaming site built to implement [this](https://blog.kunalgoel.dev/designing-youtube-s-frontend-system-streams-feeds-and-scale?utm_source=hashnode&utm_medium=feed).
I built this to understand how / what DASH was after downloading a video from a site that had an m3u8 file with lots of mpd files and to actually use some AWS services I hadn't tried.

# Stack 
Go (Gin) - nice Go http framework
sqlc - repository layer management 
PostgreSQL - metadata and user data mgt
AWS S3 - object storage of videos
AWS Lambda - thumbnail gen
ffmpeg - for chunking 

## Quickstart

Prerequisites:
- Go 1.25+
- PostgreSQL
- sqlc (optional; used to generate DB code)

1. Copy or update environment variables in `src/.env` (example):

```
DATABASE_URL=postgres://postgres:password@localhost:5432/xarg
JWT_SECRET=replace-me-for-prod
PORT=:8000
```

2. Generate sqlc code (if you change queries/schema):

```bash
sqlc generate
```

3. Build and run the server from the repo root:

```bash
cd backend
go build -o src/cmd/cmd.exe ./src/cmd
./src/cmd/cmd.exe
```

Or for development you can run:

```bash
go run ./src/cmd
```

## Configuration

- `DATABASE_URL`: Postgres connection string.
- `JWT_SECRET`: secret used to sign session JWTs (defaults to a dev secret if unset).
- `PORT`: HTTP listen address (default `:8000`).

## Endpoints

The project exposes a small set of endpoints used for health checks and auth during development.

- **GET /health**
	- Description: simple health check
	- Response: 200 JSON `{ "message": "Hello" }`

- **POST /auth/login**
	- Description: authenticate a user and return a JWT
	- Request JSON:

```json
{
	"email": "user@example.com",
	"password": "plaintext-password"
}
```

	- Response JSON (200):

```json
{
	"message": "Successfully logged in",
	"token": "<jwt>",
	"email": "user@example.com"
}
```

- **POST /auth/signup**
	- Description: create a new account and return a JWT
	- Request JSON:

```json
{
	"first_name": "Alice",
	"last_name": "Smith",
	"email": "alice@example.com",
	"password": "plaintext-password"
}
```

	- Response JSON (201):

```json
{
	"message": "Successfully signed up",
	"token": "<jwt>",
	"email": "alice@example.com",
	"firstName": "Alice"
}
```

## Notes & Next Steps

- This repo is mid-transition to `sqlc` for the repository layer. The generated code lives under `backend/src/internal/db/sqlc`.
- Authentication uses JWTs signed with `JWT_SECRET`. For production, set a strong secret and rotate as needed.
- Database migrations are not automated here — apply `backend/sql/schema.sql` to your database or integrate a migration tool.
