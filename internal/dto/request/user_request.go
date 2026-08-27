package request

//RegisterReq 注册请求
type RegisterReq struct {
	//binding:"required" 参数校验标签，required表示not null not ''
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required"`
}

//LoginReq 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"` //not null
	Password string `json:"password" binding:"required"`
}

//RefreshTokenReq 刷新token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"` //not null
}
