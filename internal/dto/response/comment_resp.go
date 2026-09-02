package response

// CommentItem 评论项返回
type CommentItem struct {
	CommentID       int64  `json:"comment_id"`
	PostID          int64  `json:"post_id"`
	ParentCommentID int64  `json:"parent_comment_id"`
	Username        string `json:"username"`
	Avatar          string `json:"avatar"`
	Content         string `json:"content"`
	IsTop           int8   `json:"is_top"`
	Status          int8   `json:"status"`
	CreatedAt       string `json:"created_at"`
}

// CreateCommentResp 创建评论成功返回
type CreateCommentResp struct {
	CommentID       int64  `json:"comment_id"`
	PostID          int64  `json:"post_id"`
	ParentCommentID int64  `json:"parent_comment_id"`
	Content         string `json:"content"`
	CreatedAt       string `json:"created_at"`
}

// UpdateCommentResp 编辑评论返回
type UpdateCommentResp struct {
	CommentID int64  `json:"comment_id"`
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}
