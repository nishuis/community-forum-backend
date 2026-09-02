package convert

import (
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
)

// ConvertPostItem 单条 domain.Post  → response.PostItem
func ConvertPostItem(po *domain.Post) *response.PostItemResp {
	if po == nil {
		return nil
	}

	item := &response.PostItemResp{
		PostID:    po.PostId,
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
		item.UserID = po.Author.UserId // ✅移到判空内部
		item.Username = po.Author.Username
		item.Avatar = po.Author.Avatar
	} else {
		//作者账号已删除
		item.UserID = 0
		item.Username = "用户已注销"
		item.Avatar = ""
	}

	return item
}

// ConvertPostItemList Post切片转响应DTO切片
func ConvertPostItemList(pos []*domain.Post) []*response.PostItemResp {
	if pos == nil {
		return []*response.PostItemResp{}
	}

	res := make([]*response.PostItemResp, 0, len(pos))
	for _, po := range pos {
		if po == nil {
			continue
		}
		res = append(res, ConvertPostItem(po))
	}

	return res
}
