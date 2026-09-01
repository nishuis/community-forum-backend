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

// ShowByKeyWordOffsetReq 分页关键词模糊查找
type ShowByKeyWordOffsetReq struct {
	KeyWord  string `json:"key_word" binding:"required"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}
