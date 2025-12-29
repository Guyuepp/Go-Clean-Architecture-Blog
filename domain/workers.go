package domain

import "context"

type LikeAction int8

const (
	Like   = 1
	Unlike = -1
)

func (l LikeAction) String() string {
	switch l {
	case Like:
		return "ADD"
	case Unlike:
		return "REMOVE"
	default:
		return "UNKNOWN"
	}
}

type SyncLikesWorker interface {
	Start(ctx context.Context)

	// Send adds a like record if action == Like, and removes a like record if action == Unlike
	Send(likeRecord UserLike, action LikeAction)
}
