package rest

import (
	"net/http"
	"strconv"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/rest/request"
	"github.com/gin-gonic/gin"
)

type commentHandler struct {
	Service domain.CommentUsecase
}

func NewCommentHandler(svc domain.CommentUsecase) *commentHandler {
	return &commentHandler{
		Service: svc,
	}
}

func (h *commentHandler) CreateComment(c *gin.Context) {
	var req request.Comment
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by authentication middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	uid := userID.(int64)
	req.UserID = uid

	// Get article ID from URL parameter
	idP, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, domain.ErrNotFound.Error())
		return
	}
	aid := int64(idP)
	req.ArticleID = aid

	comment := req.ToDomain()
	comment.UserID = userID.(int64)

	ctx := c.Request.Context()
	if err := h.Service.Create(ctx, &comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Comment created successfully", "comment": comment})
}

func (h *commentHandler) DeleteComment(c *gin.Context) {
	idP, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, domain.ErrNotFound.Error())
		return
	}
	aid := int64(idP)

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	uid := userID.(int64)

	ctx := c.Request.Context()
	if err := h.Service.Delete(ctx, aid, uid); err != nil {
		if err == domain.ErrForbidden {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to delete this comment"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}

func (h *commentHandler) FetchCommentsByArticle(c *gin.Context) {
	numS := c.Query("num")
	num, err := strconv.Atoi(numS)
	if err != nil || num < PageMinNum || num > PageMaxNum {
		num = DefaultPageNum
	}
	idP, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, domain.ErrNotFound.Error())
		return
	}
	id := int64(idP)

	cursor := c.Query("cursor")

	ctx := c.Request.Context()
	comments, nextCursor, err := h.Service.FetchByArticle(ctx, id, cursor, int64(num))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("X-cursor", nextCursor)
	c.JSON(http.StatusOK, gin.H{"comments": comments})
}
