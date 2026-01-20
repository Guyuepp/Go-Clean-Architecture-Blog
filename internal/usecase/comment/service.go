package comment

import (
	"context"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
	"github.com/Guyuepp/Go-Clean-Architecture-Blog/internal/repository"
	"github.com/sirupsen/logrus"
)

type service struct {
	commentRepo domain.CommentRepository
	bloomRepo   domain.BloomRepository
}

func (s *service) mustExists(ctx context.Context, id int64) error {
	exists, err := s.bloomRepo.Exists(ctx, id)
	if err == nil && !exists {
		logrus.Warnf("bloom filter says article %d does not exist", id)
		return domain.ErrNotFound
	}

	return nil
}

func (s *service) Create(ctx context.Context, c *domain.Comment) error {
	if err := s.mustExists(ctx, c.ArticleID); err != nil {
		if err == domain.ErrNotFound {
			return domain.ErrNotFound
		}
	}
	return s.commentRepo.Store(ctx, c)
}

func (s *service) Delete(ctx context.Context, aid int64, uid int64) error {
	return s.commentRepo.Delete(ctx, aid, uid)
}

func (s *service) FetchByArticle(ctx context.Context, articleID int64, cursor string, limit int64) ([]*domain.Comment, string, error) {
	if err := s.mustExists(ctx, articleID); err != nil {
		if err == domain.ErrNotFound {
			return nil, "", domain.ErrNotFound
		}
	}
	res, err := s.commentRepo.FetchRoots(ctx, articleID, cursor, limit)
	if err != nil {
		return []*domain.Comment{}, "", err
	}
	if len(res) == 0 {
		return []*domain.Comment{}, "", nil
	}

	rootIDs := make([]int64, len(res))
	for i, comment := range res {
		rootIDs[i] = comment.ID
	}

	replies, err := s.commentRepo.FetchReplies(ctx, rootIDs)
	if err != nil {
		return res, "", nil
	}

	replyMap := make(map[int64][]*domain.Comment)
	for _, r := range replies {
		replyMap[r.RootID] = append(replyMap[r.RootID], r)
	}

	for _, r := range res {
		if list, ok := replyMap[r.ID]; ok {
			r.Replies = list
		} else {
			r.Replies = []*domain.Comment{}
		}
	}

	return res, repository.EncodeCursor(res[len(res)-1].CreatedAt), nil
}

var _ domain.CommentUsecase = (*service)(nil)

func NewService(commentRepo domain.CommentRepository, bloomRepo domain.BloomRepository) *service {
	return &service{
		commentRepo: commentRepo,
		bloomRepo:   bloomRepo,
	}
}
