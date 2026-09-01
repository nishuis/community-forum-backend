package convert

import (
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
)

// ConvertPostListItem 单条 domain.Post  → response.PostListItem
func ConvertPostListItem(po *domain.Post) *response.PostListItem {
	if po == nil {
		return nil
	}

	item := &response.PostListItem{
		PostID:    po.PostId,
		UserID:    po.Author.UserId,
		Title:     po.Title,
		CreatedAt: po.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	//列表摘要
	const summaryMax = 100
	if len(po.Content) > summaryMax {
		item.Content = po.Content[:summaryMax] + "..."
	} else {
		item.Content = po.Content
	}

	//处理关联Author，判nil,只提取需要的两个字段，丢弃password等敏感字段
	if po.Author != nil {
		item.Username = po.Author.Username
		item.Avatar = po.Author.Avatar
	} else {
		//作者账号已删除
		item.Username = "用户已注销"
		item.Avatar = ""
	}

	return item
}

// ConvertPostListItemList Post切片转响应DTO切片
func ConvertPostListItemList(pos []*domain.Post) []*response.PostListItem {
	if pos == nil {
		return []*response.PostListItem{}
	}

	res := make([]*response.PostListItem, 0, len(pos))
	for _, po := range pos {
		res = append(res, ConvertPostListItem(po))
	}

	return res
}
