package response

//RegisterResp 注册成功返回
type RegisterResp struct {
	UserID   uint   `json:"user_id" binding:"required"`
	Username string `json:"username" binding:"required"`
}

//LoignResp 登录成功返回token
type LoginResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `kson:"refresh_token"`
}

//RefreshTokenResp 刷新token之后返回新access_token
type RefreshTokenResp struct {
	AccessToken string `json:"access_token"`
}
