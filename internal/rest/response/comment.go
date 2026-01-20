package response

import "github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"

type Comment struct {
	ID        int64  `json:"id"`
	ArticleID int64  `json:"article_id"`
	UserID    int64  `json:"user_id"`
	Content   string `json:"content"`
	ParentID  int64  `json:"parent_id"`
	RootID    int64  `json:"root_id"`
	CreatedAt string `json:"created_at"`

	// User 评论作者信息
	User *User `json:"user,omitempty"`
	// Replies 子评论列表
	Replies []*Comment `json:"replies,omitempty"`
}

func NewSingleCommentFromDomain(c *domain.Comment) *Comment {
	if c == nil {
		return nil
	}
	return &Comment{
		ID:        c.ID,
		ArticleID: c.ArticleID,
		UserID:    c.UserID,
		Content:   c.Content,
		ParentID:  c.ParentID,
		RootID:    c.RootID,
		CreatedAt: c.CreatedAt.Format(DateTimeFormat),
		User:      NewUserFromDomain(c.User),
		Replies:   nil,
	}
}

// NewCommentFromDomain: Domain -> Response
func NewCommentFromDomain(c *domain.Comment) *Comment {
	if c == nil {
		return nil
	}
	root := NewSingleCommentFromDomain(c)
	if len(c.Replies) > 0 {
		replies := make([]*Comment, 0, len(c.Replies))
		for _, r := range c.Replies {
			replies = append(replies, NewSingleCommentFromDomain(r))
		}
		root.Replies = replies
	}
	return root
}
