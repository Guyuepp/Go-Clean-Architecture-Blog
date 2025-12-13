package workers

import (
	"context"
	"log"
	"time"

	"github.com/bxcodec/go-clean-arch/domain"
	"github.com/sirupsen/logrus"
)

type SyncViewsWorker struct {
	ArticleRepo  domain.ArticleRepository
	ArticleCache domain.ArticleCache
}

func NewSyncViewWorker(ar domain.ArticleRepository, ac domain.ArticleCache) *SyncViewsWorker {
	return &SyncViewsWorker{
		ArticleRepo:  ar,
		ArticleCache: ac,
	}
}

func (s *SyncViewsWorker) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("SyncViewWorker stoped...")
				return
			default:

			}

			s.safeRun(ctx)

			time.Sleep(1 * time.Second)
			log.Println("Worker restarting...")
		}
	}()
}

func (s *SyncViewsWorker) safeRun(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("SyncViewWorker cashed(recovered): %v", err)
		}
	}()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.flush(context.Background())
			return
		case <-ticker.C:
			s.sync(context.Background())
		}
	}
}

func (s *SyncViewsWorker) syncViews(ctx context.Context) {
	views, err := s.ArticleCache.FetchAndResetViews(ctx)
	if err != nil {
		log.Printf("failed to get views from redis: %v", err)
		return
	}

	if len(views) == 0 {
		return
	}

	for id, view := range views {
		err = s.ArticleRepo.AddViews(ctx, id, view)
		if err != nil {
			logrus.Warnf("failed to update views: %v", err)
			continue
		}

	}
}

func (s *SyncViewsWorker) sync(ctx context.Context) {
	s.syncViews(ctx)
}

func (s *SyncViewsWorker) flush(ctx context.Context) {
	s.syncViews(ctx)
}
