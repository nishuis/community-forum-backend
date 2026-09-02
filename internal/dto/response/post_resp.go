package response

// CreatePostResp 发帖成功返回数据
type CreatePostResp struct {
	PostId int64  `json:"post_id"`
	Title  string `json:"title"`
}

// PostItemResp 帖子返回单元
type PostItemResp struct {
	PostID    int64  `json:"post_id"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Title     string `json:"title"`
	Content   string `json:"content"` // 列表页为摘要
	CreatedAt string `json:"created_at"`
	Avatar    string `json:"avatar"`
}
