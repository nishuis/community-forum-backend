package response

// LikeStatusResp 查询点赞状态
type LikeStatusResp struct {
	IsLike bool  `json:"is_like"`
	Count  int64 `json:"count"` //点赞总数量
}

// UserLikedItem 用户我的点赞列表项
type UserLikedItem struct {
	TargetType int8   `json:"target_type"`
	TargetID   int64  `json:"target_id"`
	CreatedAt  string `json:"created_at"`
}
