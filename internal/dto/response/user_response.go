package response

type UserResp struct {
	UserID   uint   `json:"user_id" binding:"required"`
	Username string `json:"username" binding:"required"`
}
