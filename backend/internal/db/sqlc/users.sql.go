package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createUser = `-- name: CreateUser :one
INSERT INTO users (first_name, last_name, email, password)
VALUES ($1, $2, $3, $4)
RETURNING id, email, first_name, created_at, updated_at
`

type CreateUserParams struct {
	FirstName string
	LastName  string
	Email     string
	Password  string
}

type CreateUserRow struct {
	ID        pgtype.UUID
	Email     string
	FirstName string
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error) {
	row := q.db.QueryRow(ctx, createUser,
		arg.FirstName,
		arg.LastName,
		arg.Email,
		arg.Password,
	)
	var i CreateUserRow
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.FirstName,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getUserByEmail = `-- name: GetUserByEmail :one
SELECT id, first_name, last_name, email, password, created_at, updated_at
FROM users
WHERE email = $1
  AND deleted_at IS NULL
`

type GetUserByEmailRow struct {
	ID        pgtype.UUID
	FirstName string
	LastName  string
	Email     string
	Password  string
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (GetUserByEmailRow, error) {
	row := q.db.QueryRow(ctx, getUserByEmail, email)
	var i GetUserByEmailRow
	err := row.Scan(
		&i.ID,
		&i.FirstName,
		&i.LastName,
		&i.Email,
		&i.Password,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getUserByID = `-- name: GetUserByID :one
SELECT id, email, password, created_at, updated_at
FROM users
WHERE id = $1
  AND deleted_at IS NULL
`

type GetUserByIDRow struct {
	ID        pgtype.UUID
	Email     string
	Password  string
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

func (q *Queries) GetUserByID(ctx context.Context, id pgtype.UUID) (GetUserByIDRow, error) {
	row := q.db.QueryRow(ctx, getUserByID, id)
	var i GetUserByIDRow
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Password,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
