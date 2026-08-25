package request

type RegisterReq struct {
	//binding:"required" 参数校验标签，required表示not null not ''
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
