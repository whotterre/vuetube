package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/whotterre/vuetube/src/internal/db/sqlc"
	"github.com/whotterre/vuetube/src/internal/models"
)

type UserRepository interface {
	CreateUser(ctx context.Context, userData *models.User) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserById(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type userRepository struct {
	queries *db.Queries
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{queries: db.New(pool)}
}

func (r *userRepository) CreateUser(ctx context.Context, userData *models.User) (*models.User, error) {
	createdUser, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		Email:    userData.Email,
		Password: userData.Password,
	})
	if err != nil {
		return nil, err
	}

	return mapUser(createdUser), nil
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return mapUser(user), nil
}

func (r *userRepository) GetUserById(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return mapUser(user), nil
}

func mapUser(user db.User) *models.User {
	mappedUser := &models.User{
		ID:        user.ID,
		Email:     user.Email,
		Password:  user.Password,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	if user.DeletedAt.Valid {
		deletedAt := user.DeletedAt.Time
		mappedUser.DeletedAt = &deletedAt
	}

	return mappedUser
}
