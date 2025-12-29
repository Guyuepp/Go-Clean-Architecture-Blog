package workers

import (
	"context"
	"time"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/sirupsen/logrus"
)

type LikeTask struct {
	ArticleID int64
	UserID    int64
	Action    domain.LikeAction
}

type syncLikesWorker struct {
	ArticleRepo domain.ArticleRepository
	ch          chan LikeTask
}

func NewSyncLikesWorker(ar domain.ArticleRepository) *syncLikesWorker {
	return &syncLikesWorker{
		ArticleRepo: ar,
		ch:          make(chan LikeTask, 1024),
	}
}

// Send adds a like record if action == 1, and removes a like record if action == -1
func (s syncLikesWorker) Send(likeRecord domain.UserLike, action domain.LikeAction) {
	select {
	case s.ch <- LikeTask{likeRecord.ArticleID, likeRecord.UserID, action}:
	default:
		logrus.Info("SyncLikesWorker's channel is full, task droppped")
	}
}

func (s syncLikesWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	const batchSize = 100
	batch := make([]LikeTask, 0, batchSize)
	for {
		select {
		case task := <-s.ch:
			batch = append(batch, task)
			if len(batch) == batchSize {
				s.flush(ctx, batch)
				batch = make([]LikeTask, 0, batchSize)
			}
		case <-ticker.C:
			s.flush(ctx, batch)
			batch = make([]LikeTask, 0)
		case <-ctx.Done():
			logrus.Info("shuting down SyncLikesWorker, flushing remain tasks...")
			s.flush(ctx, batch)
		}
	}
}

type taskKey struct {
	aid, uid int64
}

func (s syncLikesWorker) flush(ctx context.Context, batch []LikeTask) {
	tasks := make(map[taskKey]domain.LikeAction)
	for i := range batch {
		key := taskKey{
			aid: batch[i].ArticleID,
			uid: batch[i].UserID,
		}
		tasks[key] = batch[i].Action
	}
	var changes domain.LikeStateChanges
	for key, action := range tasks {
		switch action {
		case domain.Like:
			changes.ToAdd = append(changes.ToAdd, domain.UserLike{
				ArticleID: key.aid,
				UserID:    key.uid,
			})
		case domain.Unlike:
			changes.ToRemove = append(changes.ToRemove, domain.UserLike{
				ArticleID: key.aid,
				UserID:    key.uid,
			})
		default:
			logrus.Errorf("Unsuported action: %v", action)
		}
	}
	_ = s.ArticleRepo.ApplyLikeChanges(ctx, changes)
}
