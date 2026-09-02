package convert

import (
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
)

// ConvertCommentItem domain.Comment -> response.CommentItem
func ConvertCommentItem(po *domain.Comment) *response.CommentItem {
	item := &response.CommentItem{
		CommentID:       po.CommentID,
		PostID:          po.PostID,
		ParentCommentID: po.ParentCommentID,
		Content:         po.Content,
		IsTop:           po.IsTop,
		Status:          po.Status,
		CreatedAt:       po.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	// 用户注销，Author预加载为nil，展示占位文字
	if po.Author != nil {
		item.Username = po.Author.Username
		item.Avatar = po.Author.Avatar
	} else {
		item.Username = "用户已注销"
		item.Avatar = ""
	}
	return item
}

// ConvertCommentItemList []*domain.Comment -> []*response.CommentItem
func ConvertCommentItemList(pos []*domain.Comment) []*response.CommentItem {
	if pos == nil {
		return []*response.CommentItem{}
	}

	res := make([]*response.CommentItem, 0, len(pos))
	for _, v := range pos {
		res = append(res, ConvertCommentItem(v))
	}

	return res
}

// ConvertCreateCommentResp 创建评论返回
func ConvertCreateCommentResp(po *domain.Comment) *response.CreateCommentResp {
	return &response.CreateCommentResp{
		CommentID:       po.CommentID,
		PostID:          po.PostID,
		ParentCommentID: po.ParentCommentID,
		Content:         po.Content,
		CreatedAt:       po.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}

// ConvertEditCommentResp 编辑评论响应转换
func ConvertEditCommentResp(po *domain.Comment) *response.UpdateCommentResp {
	return &response.UpdateCommentResp{
		CommentID: po.CommentID,
		Content:   po.Content,
		UpdatedAt: po.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
