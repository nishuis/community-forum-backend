package convert

import (
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
)

func ConvertGetCurrentUserResp(user *domain.User) *response.GetCurrentUserResp {
	if user == nil {
		return nil
	}
	resp := &response.GetCurrentUserResp{
		UserID:   user.UserId,
		Username: user.Username,
		Email:    user.Email,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
	}
	return resp
}

// ConvertUserItem domain.User → response.UserItemResp
func ConvertUserItem(u *domain.User) *response.UserItemResp {
	if u == nil {
		return nil
	}
	item := &response.UserItemResp{
		UserId:   u.UserId,
		Username: u.Username,
		Avatar:   u.Avatar,
	}
	return item
}

// ConvertUserItemList User实体切片转响应切片
func ConvertUserItemList(users []*domain.User) []*response.UserItemResp {
	if users == nil {
		return []*response.UserItemResp{}
	}
	res := make([]*response.UserItemResp, 0, len(users))
	for _, u := range users {
		if u == nil {
			continue
		}
		res = append(res, ConvertUserItem(u))
	}
	return res
}
