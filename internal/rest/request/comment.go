package request

import "github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"

type Comment struct {
	ID        int64  `json:"id"`                         // for DEL
	ArticleID int64  `json:"article_id"`                 // for CREATE
	UserID    int64  `json:"user_id"`                    // for CREATE
	Content   string `json:"content" binding:"required"` // for CREATE
	ParentID  int64  `json:"parent_id"`                  // for CREATE
	RootID    int64  `json:"root_id"`                    // for CREATE
}

// ToDomain: Request -> Domain
func (r *Comment) ToDomain() domain.Comment {
	return domain.Comment{
		ID:        r.ID,
		ArticleID: r.ArticleID,
		UserID:    r.UserID,
		Content:   r.Content,
		ParentID:  r.ParentID,
		RootID:    r.RootID,
	}
}
