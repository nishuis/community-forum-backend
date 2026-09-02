package convert

import (
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
)

// ConvertUserLikedItem domain.Like -> response.UserLikedItem
func ConvertUserLikedItem(po *domain.Like) *response.UserLikedItem {
	return &response.UserLikedItem{
		TargetType: po.TargetType,
		TargetID:   po.TargetID,
		CreatedAt:  po.CreatedAt.Format("2006‑01‑02 15:04:05"),
	}
}

// ConvertUserLikedItemList []*domain.Like -> []*response.UserLikedItem
func ConvertUserLikedItemList(pos []*domain.Like) []*response.UserLikedItem {
	if pos == nil {
		return []*response.UserLikedItem{}
	}
	res := make([]*response.UserLikedItem, 0, len(pos))
	for _, v := range pos {
		res = append(res, ConvertUserLikedItem(v))
	}
	return res
}
