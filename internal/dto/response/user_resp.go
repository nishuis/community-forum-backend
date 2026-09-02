package response

//RegisterResp 注册响应
type RegisterResp struct {
	UserID   int64  `json:"user_id" binding:"required"`
	Username string `json:"username" binding:"required"`
}

//LoignResp 登录响应
type LoginResp struct {
	Username     string `json:"username"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

//RefreshTokenResp 刷新token响应
type RefreshTokenResp struct {
	AccessToken string `json:"access_token"`
}

// GetCurrentUserResp 返回用户信息
type GetCurrentUserResp struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Nickname string `json:"nick_name"`
	Avatar   string `json:"avatar"`
}

// UserItemResp 标准公开用户单元响应，帖子/评论作者展示，脱敏，不含密码、邮箱
type UserItemResp struct {
	UserId   int64  `json:"user_id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}
