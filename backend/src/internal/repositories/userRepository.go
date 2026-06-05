package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/whotterre/vuetube/src/internal/db/sqlc"
)

type UserRepository interface {
	CreateUser(ctx context.Context, userData db.CreateUserParams) (*db.User, error)
	GetUserByEmail(ctx context.Context, email string) (*db.User, error)
	GetUserById(ctx context.Context, id uuid.UUID) (*db.User, error)
}

type userRepository struct {
	queries *db.Queries
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{queries: db.New(pool)}
}

func (r *userRepository) CreateUser(ctx context.Context, params db.CreateUserParams) (*db.User, error) {
	createdUser, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		return nil, err
	}

	res := &db.User{
		ID:        createdUser.ID,
		Email:     createdUser.Email,
		FirstName: createdUser.FirstName,
		LastName:  createdUser.LastName,
		Country:   createdUser.Country,
		CreatedAt: createdUser.CreatedAt,
		UpdatedAt: createdUser.UpdatedAt,
	}
	return res, nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*db.User, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	res := &db.User{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Country:   user.Country,
		Email:     user.Email,
		Password:  user.Password,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	return res, nil
}

func (r *userRepository) GetUserById(ctx context.Context, id uuid.UUID) (*db.User, error) {
	user, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	res := &db.User{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Country:   user.Country,
		Password:  user.Password,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
	return res, nil
}
