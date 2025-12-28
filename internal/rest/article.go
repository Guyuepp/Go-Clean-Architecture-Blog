package rest

import (
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

// ArticleHandler  represent the httphandler for article
type ArticleHandler struct {
	Service domain.ArticleUsecase
}

const (
	DefaultPageNum = 10
	PageMinNum     = 5
	PageMaxNum     = 30

	DefaultRankLimit = 10
	RankMin          = 5
	RankMax          = 30
)

func NewArticleHandler(svc domain.ArticleUsecase) *ArticleHandler {
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
	if err != nil || num < PageMinNum || num > PageMaxNum {
		num = DefaultPageNum
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

// Like adds a like record if not exists
func (a *ArticleHandler) Like(c *gin.Context) {
	idP, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, domain.ErrNotFound.Error())
		return
	}
	aid := int64(idP)
	UserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	uid := UserID.(int64)
	ok, err := a.Service.AddLikeRecord(c.Request.Context(), domain.UserLike{
		ArticleID: aid,
		UserID:    uid,
	})
	if err != nil {
		c.JSON(getStatusCode(err), ResponseError{err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_changed": ok})
}

// Unlike removes a like record if exists
func (a *ArticleHandler) Unlike(c *gin.Context) {
	idP, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, domain.ErrNotFound.Error())
		return
	}
	aid := int64(idP)
	UserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	uid := UserID.(int64)
	ok, err := a.Service.RemoveLikeRecord(c.Request.Context(), domain.UserLike{
		ArticleID: aid,
		UserID:    uid,
	})
	if err != nil {
		c.JSON(getStatusCode(err), ResponseError{err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_changed": ok})
}

func (a *ArticleHandler) FetchRank(c *gin.Context) {
	limitS := c.Query("limit")
	limit, err := strconv.ParseInt(limitS, 10, 64)
	if err != nil || limit < RankMin || limit > RankMax {
		limit = DefaultRankLimit
		logrus.Error("Invalid param 'limit'")
	}
	rankType := c.DefaultQuery("type", "daily")

	var listAr []domain.Article

	switch rankType {
	case "daily":
		listAr, err = a.Service.FetchDailyRank(c.Request.Context(), limit)
	case "history":
		listAr, err = a.Service.FetchHistoryRank(c.Request.Context(), limit)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rank type"})
		return
	}
	if err != nil {
		c.JSON(getStatusCode(err), ResponseError{err.Error()})
		return
	}

	res := make([]response.Article, len(listAr))
	for i := range listAr {
		res[i] = response.NewArticleFromDomain(&listAr[i])
	}
	c.JSON(http.StatusOK, res)
}

// getStatusCode will get the code of the error from domain.ArticleUsecase
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
