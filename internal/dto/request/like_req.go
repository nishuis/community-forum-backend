package request

// DoLikeReq 执行点赞请求
type DoLikeReq struct {
	TargetType int8  `json:"target_type" binding:"required,oneof=1 2"` //1帖子 2评论
	TargetID   int64 `json:"target_id" binding:"required,min=1"`
}

// CancelLikeReq 取消点赞请求
type CancelLikeReq struct {
	TargetType int8  `json:"target_type" binding:"required,oneof=1 2"`
	TargetID   int64 `json:"target_id" binding:"required,min=1"`
}
