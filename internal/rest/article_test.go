package rest_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	faker "github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/bxcodec/go-clean-arch/internal/rest"
	"github.com/bxcodec/go-clean-arch/internal/rest/mocks"
)

func setupRouterWithArticleHandler(svc rest.ArticleService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	rest.NewArticleHandler(r, svc)
	return r
}

func TestFetch(t *testing.T) {
	var mockArticle domain.Article
	err := faker.FakeData(&mockArticle)
	require.NoError(t, err)

	mockSvc := new(mocks.ArticleService)
	mockList := []domain.Article{mockArticle}
	num := 1
	cursor := "2"

	mockSvc.
		On("Fetch", mock.Anything, cursor, int64(num)).
		Return(mockList, "10", nil)

	r := setupRouterWithArticleHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/articles?num=1&cursor="+cursor, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "10", rec.Header().Get("X-cursor"))

	var got []domain.Article
	err = json.Unmarshal(rec.Body.Bytes(), &got)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, mockArticle.ID, got[0].ID)

	mockSvc.AssertExpectations(t)
}

func TestFetchError(t *testing.T) {
	mockSvc := new(mocks.ArticleService)
	num := 1
	cursor := "2"

	mockSvc.
		On("Fetch", mock.Anything, cursor, int64(num)).
		Return(nil, "", domain.ErrInternalServerError)

	r := setupRouterWithArticleHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/articles?num=1&cursor="+cursor, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "", rec.Header().Get("X-cursor"))

	var resp rest.ResponseError
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Message)

	mockSvc.AssertExpectations(t)
}

func TestGetByID(t *testing.T) {
	var mockArticle domain.Article
	err := faker.FakeData(&mockArticle)
	require.NoError(t, err)

	mockSvc := new(mocks.ArticleService)
	id := int64(mockArticle.ID)

	mockSvc.
		On("GetByID", mock.Anything, id).
		Return(mockArticle, nil)

	r := setupRouterWithArticleHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/articles/"+strconv.FormatInt(id, 10), nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var got domain.Article
	err = json.Unmarshal(rec.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, mockArticle.ID, got.ID)

	mockSvc.AssertExpectations(t)
}

func TestStore(t *testing.T) {
	mockArticle := domain.Article{
		Title:     "Title",
		Content:   "Content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockSvc := new(mocks.ArticleService)

	// 这里期望传入任意 *domain.Article 即可
	mockSvc.
		On("Store", mock.Anything, mock.AnythingOfType("*domain.Article")).
		Return(nil)

	r := setupRouterWithArticleHandler(mockSvc)

	bodyBytes, err := json.Marshal(mockArticle)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/articles", strings.NewReader(string(bodyBytes)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var got domain.Article
	err = json.Unmarshal(rec.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, mockArticle.Title, got.Title)

	mockSvc.AssertExpectations(t)
}

func TestDelete(t *testing.T) {
	var mockArticle domain.Article
	err := faker.FakeData(&mockArticle)
	require.NoError(t, err)

	mockSvc := new(mocks.ArticleService)
	id := int64(mockArticle.ID)

	mockSvc.
		On("Delete", mock.Anything, id).
		Return(nil)

	r := setupRouterWithArticleHandler(mockSvc)

	req := httptest.NewRequest(http.MethodDelete, "/articles/"+strconv.FormatInt(id, 10), nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())

	mockSvc.AssertExpectations(t)
}
