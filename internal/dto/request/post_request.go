package request

// CreatePostReq 发帖请求体
type CreatePostReq struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// UpdatePostReq 编辑帖子
type UpdatePostReq struct {
	UpdateTitle   string `json:"update_title"`
	UpdateContent string `json:"update_content"`
}
