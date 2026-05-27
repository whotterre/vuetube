package db

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	ID        pgtype.UUID
	Email     string
	FirstName string
	LastName  string
	Password  string
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
	DeletedAt pgtype.Timestamptz
}
