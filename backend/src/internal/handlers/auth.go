package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/whotterre/vuetube/src/dto"
	"github.com/whotterre/vuetube/src/internal/services"
)

type AuthHandler interface {
	Login(ctx *gin.Context)
	Signup(ctx *gin.Context)
}

type authHandler struct {
	authService services.UserService
}

func NewAuthHandler(authService services.UserService) AuthHandler {
	return &authHandler{
		authService: authService,
	}
}

func (h *authHandler) Login(ctx *gin.Context) {
	var req dto.LoginUserDto
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Failed to read request body",
		})
		return
	}
	res, err := h.authService.LoginUser(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Successfully logged in",
		"token":   res.Token,
		"email":   res.Email,
	})
}

func (h *authHandler) Signup(ctx *gin.Context) {
	var req dto.SignupUserRequestDto
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Failed to read request body during signup",
		})
		return
	}
	if req.Country == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "country is required",
		})
		return
	}
	res, err := h.authService.SignupUser(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{
		"message":   "Successfully signed up",
		"token":     res.Token,
		"email":     res.Email,
		"firstName": res.FirstName,
		"country":   res.Country,
	})
}
