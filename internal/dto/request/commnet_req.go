package request

// CreateCommentReq 发表评论请求
type CreateCommentReq struct {
	PostID          int64  `json:"post_id" binding:"required"`
	ParentCommentID int64  `json:"parent_comment_id"` // 一级评论 0 ，回复评论填对应id
	Content         string `json:"content" binding:"required,max=500"`
}

// UpdateCommentReq 编辑评论请求
type UpdateCommentReq struct {
	CommentID int64  `json:"comment_id" binding:"required"`
	Content   string `json:"content" binding:"required,max=500"`
}
