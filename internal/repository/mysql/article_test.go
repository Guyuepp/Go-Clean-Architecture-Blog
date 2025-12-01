package mysql_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/bxcodec/go-clean-arch/internal/repository"
	articleMysqlRepo "github.com/bxcodec/go-clean-arch/internal/repository/mysql"
)

func TestFetchArticle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	mockArticles := []domain.Article{
		{
			ID: 1, Title: "title 1", Content: "content 1",
			Author: domain.Author{ID: 1}, UpdatedAt: time.Now(), CreatedAt: time.Now(), Views: 0,
		},
		{
			ID: 2, Title: "title 2", Content: "content 2",
			Author: domain.Author{ID: 1}, UpdatedAt: time.Now(), CreatedAt: time.Now(), Views: 0,
		},
	}

	rows := sqlmock.NewRows([]string{"id", "title", "content", "author_id", "updated_at", "created_at", "views"}).
		AddRow(mockArticles[0].ID, mockArticles[0].Title, mockArticles[0].Content,
			mockArticles[0].Author.ID, mockArticles[0].UpdatedAt, mockArticles[0].CreatedAt, mockArticles[0].Views).
		AddRow(mockArticles[1].ID, mockArticles[1].Title, mockArticles[1].Content,
			mockArticles[1].Author.ID, mockArticles[1].UpdatedAt, mockArticles[1].CreatedAt, mockArticles[1].Views)

	query := "SELECT id,title,content, author_id, updated_at, created_at, views FROM article WHERE created_at > \\? ORDER BY created_at LIMIT \\?"

	mock.ExpectQuery(query).WillReturnRows(rows)
	a := articleMysqlRepo.NewArticleRepository(db)
	cursor := repository.EncodeCursor(mockArticles[1].CreatedAt)
	num := int64(2)
	list, nextCursor, err := a.Fetch(context.TODO(), cursor, num)
	assert.NotEmpty(t, nextCursor)
	assert.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestGetArticleByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	rows := sqlmock.NewRows([]string{"id", "title", "content", "author_id", "updated_at", "created_at", "views"}).
		AddRow(1, "title 1", "Content 1", 1, time.Now(), time.Now(), 0)

	query := "SELECT id,title,content, author_id, updated_at, created_at, views FROM article WHERE ID = \\?"

	mock.ExpectQuery(query).WillReturnRows(rows)
	a := articleMysqlRepo.NewArticleRepository(db)

	num := int64(5)
	anArticle, err := a.GetByID(context.TODO(), num)
	assert.NoError(t, err)
	assert.NotNil(t, anArticle)
}

func TestStoreArticle(t *testing.T) {
	now := time.Now()
	ar := &domain.Article{
		Title:     "Judul",
		Content:   "Content",
		CreatedAt: now,
		UpdatedAt: now,
		Author: domain.Author{
			ID:   1,
			Name: "Iman Tumorang",
		},
		Views: 0,
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	query := "INSERT  article SET title=\\? , content=\\? , author_id=\\?, updated_at=\\? , created_at=\\?, views=\\?"
	prep := mock.ExpectPrepare(query)
	prep.ExpectExec().WithArgs(ar.Title, ar.Content, ar.Author.ID, ar.CreatedAt, ar.UpdatedAt, ar.Views).WillReturnResult(sqlmock.NewResult(12, 1))

	a := articleMysqlRepo.NewArticleRepository(db)

	err = a.Store(context.TODO(), ar)
	assert.NoError(t, err)
	assert.Equal(t, int64(12), ar.ID)
}

func TestGetArticleByTitle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	rows := sqlmock.NewRows([]string{"id", "title", "content", "author_id", "updated_at", "created_at", "views"}).
		AddRow(1, "title 1", "Content 1", 1, time.Now(), time.Now(), 0)

	query := "SELECT id,title,content, author_id, updated_at, created_at, views FROM article WHERE title = \\?"

	mock.ExpectQuery(query).WillReturnRows(rows)
	a := articleMysqlRepo.NewArticleRepository(db)

	title := "title 1"
	anArticle, err := a.GetByTitle(context.TODO(), title)
	assert.NoError(t, err)
	assert.NotNil(t, anArticle)
}

func TestDeleteArticle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	query := "DELETE FROM article WHERE id = \\?"

	prep := mock.ExpectPrepare(query)
	prep.ExpectExec().WithArgs(12).WillReturnResult(sqlmock.NewResult(12, 1))

	a := articleMysqlRepo.NewArticleRepository(db)

	num := int64(12)
	err = a.Delete(context.TODO(), num)
	assert.NoError(t, err)
}

func TestUpdateArticle(t *testing.T) {
	now := time.Now()
	ar := &domain.Article{
		ID:        12,
		Title:     "Judul",
		Content:   "Content",
		CreatedAt: now,
		UpdatedAt: now,
		Author: domain.Author{
			ID:   1,
			Name: "Iman Tumorang",
		},
		Views: 0,
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	query := "UPDATE article set title=\\?, content=\\?, author_id=\\?, updated_at=\\? WHERE ID = \\?"

	prep := mock.ExpectPrepare(query)
	prep.ExpectExec().WithArgs(ar.Title, ar.Content, ar.Author.ID, ar.UpdatedAt, ar.ID).WillReturnResult(sqlmock.NewResult(12, 1))

	a := articleMysqlRepo.NewArticleRepository(db)

	err = a.Update(context.TODO(), ar)
	assert.NoError(t, err)
}

// ...existing code...
func TestIncreaseViews(t *testing.T) {
	// 1. 准备测试数据
	mockArticleID := int64(12)

	// 2. 初始化 Mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	// 记得关闭假连接
	defer db.Close()

	// 3. 设置期望
	// 正则表达式匹配 SQL。注意：
	// - `+` 在正则里有特殊含义，所以要写成 `\+`
	// - `?` 也要转义 `\?`
	query := "UPDATE article SET views=views\\+1 WHERE id=\\?"

	// 因为代码里直接用了 ExecContext，所以这里直接用 ExpectExec
	// 如果代码里用了 Prepare，这里就要先 ExpectPrepare 再 ExpectExec
	mock.ExpectExec(query).
		WithArgs(mockArticleID).
		WillReturnResult(sqlmock.NewResult(0, 1)) // 模拟返回：LastInsertId=0, RowsAffected=1

	// 4. 初始化你的 Repository
	a := articleMysqlRepo.NewArticleRepository(db)

	// 5. 执行代码
	err = a.IncreaseViews(context.TODO(), mockArticleID)

	// 6. 验证结果
	assert.NoError(t, err)

	// 验证所有的 mock 期望是否都已满足（这是一个好习惯）
	assert.NoError(t, mock.ExpectationsWereMet())
}
