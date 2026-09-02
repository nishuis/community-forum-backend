package repository

import (
	"context"

	"github.com/nishuis/community-forum-backend/internal/domain"
	"gorm.io/gorm"
)

type CommentRepo struct {
	db *gorm.DB
}

func NewCommentRepo(db *gorm.DB) *CommentRepo {
	return &CommentRepo{db: db}
}

// // CreateComment 创建评论
func (r *CommentRepo) CreateComment(ctx context.Context, comment *domain.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

// DeleteComment 软删除评论，传入commentId
func (r *CommentRepo) DeleteComment(ctx context.Context, commentId int64) error {
	return r.db.WithContext(ctx).
		Where("comment_id = ?", commentId).
		Delete(&domain.Comment{}).Error
}

// FindCommentByID 根据评论ID查询单条评论，预加载作者Author
// gorm.ErrRecordNotFound 评论不存在
func (r *CommentRepo) FindCommentByID(ctx context.Context, commentId int64) (*domain.Comment, error) {
	var comment domain.Comment
	err := r.db.WithContext(ctx).
		Preload("Author").
		Where("comment_id = ?", commentId).
		First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// FindCommentPageByPostID 根据帖子ID分页查询正常评论 status=0
// 排序规则：置顶优先，再按创建时间倒序；预加载Author
// 返回：评论列表、总条数、error
func (r *CommentRepo) FindCommentPageByPostID(ctx context.Context, postId int64, page int, pageSize int) ([]*domain.Comment, int64, error) {
	var list []*domain.Comment
	var total int64

	db := r.db.WithContext(ctx).Model(&domain.Comment{}).
		Preload("Author").
		Where("post_id = ? AND status = 0", postId)

	//统计总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := db.Order("is_top DESC, created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error

	return list, total, err
}
