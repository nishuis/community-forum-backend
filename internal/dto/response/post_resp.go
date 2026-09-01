package response

// CreatePostResp 发帖成功返回数据
type CreatePostResp struct {
	PostId int64  `json:"post_id"`
	Title  string `json:"title"`
}

// PostListItem 帖子列表每一项返回结构
type PostListItem struct {
	PostID    int64  `json:"post_id"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Title     string `json:"title"`
	Content   string `json:"content"` // 列表页为摘要
	CreatedAt string `json:"created_at"`
	Avatar    string `json:"avatar"`
}
