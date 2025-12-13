package rest

import (
	"context"
	"net/http"
	"strconv"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/bxcodec/go-clean-arch/internal/rest/request"
	"github.com/bxcodec/go-clean-arch/internal/rest/response"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ResponseError represent the response error struct
type ResponseError struct {
	Message string `json:"message"`
}

//go:generate mockery --name ArticleService
type ArticleService interface {
	Fetch(ctx context.Context, cursor string, num int64) ([]domain.Article, string, error)
	GetByID(ctx context.Context, id int64) (domain.Article, error)
	Update(ctx context.Context, ar *domain.Article) error
	UpdateViews(ctx context.Context, id int64, newViews int64) error
	GetByTitle(ctx context.Context, title string) (domain.Article, error)
	Store(context.Context, *domain.Article) error
	Delete(ctx context.Context, id int64) error
}

// ArticleHandler  represent the httphandler for article
type ArticleHandler struct {
	Service ArticleService
}

const defaultNum = 10

func NewArticleHandler(svc ArticleService) *ArticleHandler {
	return &ArticleHandler{
		Service: svc,
	}
}

// GetByID will get article by given id
func (a *ArticleHandler) GetByID(c *gin.Context) {
	idP, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, ResponseError{Message: domain.ErrNotFound.Error()})
		return
	}
	id := int64(idP)
	ctx := c.Request.Context()

	art, err := a.Service.GetByID(ctx, id)
	if err != nil {
		c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, response.NewArticleFromDomain(&art))
}

// FetchArticle will fetch the articles based on given params
func (a *ArticleHandler) FetchArticle(c *gin.Context) {
	numS := c.Query("num")
	num, err := strconv.Atoi(numS)
	if err != nil || num == 0 {
		num = defaultNum
		logrus.Error("Invalid param 'num'")
	}

	cursor := c.Query("cursor")
	ctx := c.Request.Context()

	listAr, nextCursor, err := a.Service.Fetch(ctx, cursor, int64(num))
	if err != nil {
		c.JSON(getStatusCode(err), ResponseError{Message: err.Error()})
		return
	}
	res := make([]response.Article, len(listAr))
	for i := range listAr {
		res[i] = response.NewArticleFromDomain(&listAr[i])
	}
	c.Header(`X-cursor`, nextCursor)
	c.JSON(http.StatusOK, res)
}

// Store will store the article by given request body
func (a *ArticleHandler) Store(c *gin.Context) {
	var req request.Article
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	article := req.ToDomain()
	article.User.ID = userID.(int64)

	// 4. 调用 Service
	ctx := c.Request.Context()
	if err := a.Service.Store(ctx, &article); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response.NewArticleFromDomain(&article))
}

// Delete will delete the article by given param
func (a *ArticleHandler) Delete(c *gin.Context) {
	idP, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, domain.ErrNotFound.Error())
		return
	}
	id := int64(idP)

	if err := a.Service.Delete(c.Request.Context(), id); err != nil {
		c.JSON(getStatusCode(err), ResponseError{err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// getStatusCode will get the code of the error from ArticleService
func getStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	logrus.Error(err)
	switch err {
	case domain.ErrInternalServerError:
		return http.StatusInternalServerError
	case domain.ErrNotFound:
		return http.StatusNotFound
	case domain.ErrConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
