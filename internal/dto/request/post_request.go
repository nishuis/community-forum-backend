package request

// CreatePostReq 发帖请求体
type CreatePostReq struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}
