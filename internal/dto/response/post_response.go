package response

// CreatePostResp 发帖成功返回数据
type CreatePostResp struct {
	PostId int64  `json:"post_id"`
	Title  string `json:"title"`
}
