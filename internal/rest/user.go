package rest

import (
	"context"
	"net/http"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/bxcodec/go-clean-arch/internal/rest/request"
	"github.com/gin-gonic/gin"
)

type UserService interface {
	Register(ctx context.Context, name, username, password string) error
	Login(ctx context.Context, username, password string) (string, error)
	EditPassword(ctx context.Context, id int64, oldPassword, newPassword string) error
}

type UserHandler struct {
	Service UserService
}

func NewUserHandler(svc UserService) *UserHandler {
	return &UserHandler{
		Service: svc,
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var req request.User

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.Service.Register(c.Request.Context(), req.Name, req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

// Login handles user login and returns a JWT token upon successful authentication
func (h *UserHandler) Login(c *gin.Context) {
	var req request.User

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.Service.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if err == domain.ErrBadParamInput || err == domain.ErrNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
