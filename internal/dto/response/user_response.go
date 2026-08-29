package response

//RegisterResp 注册响应
type RegisterResp struct {
	UserID   int64  `json:"user_id" binding:"required"`
	Username string `json:"username" binding:"required"`
}

//LoignResp 登录响应
type LoginResp struct {
	UserID       int64  `json:"user_id"`
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
