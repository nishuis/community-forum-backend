package convert

import (
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
)

// ConvertUserLikedItem domain.Like -> response.UserLikedItem
func ConvertUserLikedItem(po *domain.Like) *response.UserLikedItemResp {
	return &response.UserLikedItemResp{
		TargetType: po.TargetType,
		TargetID:   po.TargetID,
		CreatedAt:  po.CreatedAt.Format("2006‑01‑02 15:04:05"),
	}
}

// ConvertUserLikedItemList []*domain.Like -> []*response.UserLikedItem
func ConvertUserLikedItemList(pos []*domain.Like) []*response.UserLikedItemResp {
	if pos == nil {
		return []*response.UserLikedItemResp{}
	}
	res := make([]*response.UserLikedItemResp, 0, len(pos))
	for _, v := range pos {
		if v == nil {
			continue
		}
		res = append(res, ConvertUserLikedItem(v))
	}
	return res
}
