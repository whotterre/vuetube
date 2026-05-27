package services

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/whotterre/vuetube/src/dto"
	db "github.com/whotterre/vuetube/src/internal/db/sqlc"
	"github.com/whotterre/vuetube/src/internal/repositories"
	"github.com/whotterre/vuetube/src/internal/utils"
)

type UserService interface {
	LoginUser(ctx *gin.Context, loginData dto.LoginUserDto) (*dto.LoginUserResponseDto, error)
	SignupUser(ctx *gin.Context, signUpData dto.SignupUserRequestDto) (*dto.SignupResponseDto, error)
}

type userService struct {
	userRepo  repositories.UserRepository
	jwtSecret string
}

func NewUserService(userRepo repositories.UserRepository, jwtSecret string) UserService {
	return &userService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
	}
}

func (s *userService) LoginUser(ctx *gin.Context, loginData dto.LoginUserDto) (*dto.LoginUserResponseDto, error) {
	user, err := s.userRepo.GetUserByEmail(context.Background(), loginData.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginData.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	token, err := utils.GenerateToken(user.ID, user.Email, s.jwtSecret, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &dto.LoginUserResponseDto{
		Token: token,
		Email: user.Email,
	}, nil
}

func (s *userService) SignupUser(ctx *gin.Context, signUpData dto.SignupUserRequestDto) (*dto.SignupResponseDto, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(signUpData.Password), 10)
	if err != nil {
		return nil, err
	}

	newUser := db.CreateUserParams{
		Email:     signUpData.Email,
		Password:  string(hashedPassword),
		FirstName: signUpData.FirstName,
		LastName:  signUpData.LastName,
	}

	createdUser, err := s.userRepo.CreateUser(context.Background(), newUser)
	if err != nil {
		return nil, err
	}

	token, err := utils.GenerateToken(createdUser.ID, createdUser.Email, s.jwtSecret, 1*time.Hour)
	if err != nil {
		return nil, err
	}

	return &dto.SignupResponseDto{
		Token:     token,
		Email:     createdUser.Email,
		FirstName: createdUser.FirstName,
	}, nil
}
