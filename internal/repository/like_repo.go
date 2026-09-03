package repository

import (
	"context"

	"github.com/nishuis/community-forum-backend/internal/domain"
	"gorm.io/gorm"
)

type LikeRepo struct {
	db *gorm.DB
}

func NewLikeRepo(db *gorm.DB) *LikeRepo {
	return &LikeRepo{db: db}
}

// CreateLike 创建点赞记录,不更新计数器
// 唯一索引冲突返回 gorm.ErrDuplicatedKey，上层service捕获转为业务错误
func (r *LikeRepo) CreateLike(ctx context.Context, like *domain.Like) error {
	return r.db.WithContext(ctx).Create(like).Error
}

// DoPostLikeTx 帖子点赞：事务内插入like + post like_count +1
// 冲突返回 gorm.ErrDuplicatedKey（数据库唯一索引兜底）
func (r *LikeRepo) DoPostLikeTx(ctx context.Context, userID int64, postID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		like := &domain.Like{
			UserID:     userID,
			TargetType: domain.LikeTargetTypePost,
			TargetID:   postID,
		}
		if err := tx.Create(like).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Post{}).
			Where("post_id = ?", postID).
			Update("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
			return err
		}
		return nil
	})
}

// DoCommentLikeTx 评论点赞：事务内插入like + comment like_count +1
// 冲突返回 gorm.ErrDuplicatedKey
func (r *LikeRepo) DoCommentLikeTx(ctx context.Context, userID int64, commentID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		like := &domain.Like{
			UserID:     userID,
			TargetType: domain.LikeTargetTypeComment,
			TargetID:   commentID,
		}
		if err := tx.Create(like).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.Comment{}).
			Where("comment_id = ?", commentID).
			Update("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
			return err
		}
		return nil
	})
}

// CancelLike 取消点赞，物理删除，不更新计数器
func (r *LikeRepo) CancelLike(ctx context.Context, userID int64, targetType int8, targetID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Delete(&domain.Like{}).Error
}

// CancelPostLikeTx 帖子取消点赞：事务内删除like + post like_count -1
// 无记录可删时Delete不会报错，受影响行数为0；更新条件 like_count>0 保护不出现负数
func (r *LikeRepo) CancelPostLikeTx(ctx context.Context, userID int64, postID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除点赞记录
		err := tx.Where("user_id = ? AND target_type = ? AND target_id = ?",
			userID, domain.LikeTargetTypePost, postID).
			Delete(&domain.Like{}).Error
		if err != nil {
			return err
		}

		// 原子减1，like_count>0防止负数
		err = tx.Model(&domain.Post{}).
			Where("post_id = ? AND like_count > 0", postID).
			Update("like_count", gorm.Expr("like_count - ?", 1)).Error
		return err
	})
}

// CancelCommentLikeTx 评论取消点赞：事务内删除like + comment like_count -1
func (r *LikeRepo) CancelCommentLikeTx(ctx context.Context, userID int64, commentID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("user_id = ? AND target_type = ? AND target_id = ?",
			userID, domain.LikeTargetTypeComment, commentID).
			Delete(&domain.Like{}).Error
		if err != nil {
			return err
		}

		err = tx.Model(&domain.Comment{}).
			Where("comment_id = ? AND like_count > 0", commentID).
			Update("like_count", gorm.Expr("like_count - ?", 1)).Error
		return err
	})
}

// IsUserLike 判断用户是否对目标已点赞（只查询未软删除记录）
// 返回：是否点赞，error
func (r *LikeRepo) IsUserLike(ctx context.Context, userID int64, targetType int8, targetID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Like{}).
		Where("user_id = ? AND target_type = ? AND target_id = ?", userID, targetType, targetID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CountLike 统计目标点赞总数，只统计有效未取消点赞
func (r *LikeRepo) CountLike(ctx context.Context, targetType int8, targetID int64) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Like{}).
		Where("target_type = ? AND target_id = ?", targetType, targetID).
		Count(&total).Error
	return total, err
}

// BatchGetUserLikeIds 接收一个id集合，批量返回这批id中，被用户点过赞的id，优于N+1方案
// targetType: 1帖子 /2评论；targetIds:一批目标id
// 返回用户真实点赞的targetId切片
func (r *LikeRepo) BatchGetUserLikeIds(ctx context.Context, userID int64, targetType int8, targetIds []int64) ([]int64, error) {
	var res []int64
	err := r.db.WithContext(ctx).Model(&domain.Like{}).
		Select("target_id").
		Where("user_id = ? AND target_type = ? AND target_id IN ?", userID, targetType, targetIds).
		Find(&res).Error
	return res, err
}

// FindUserLikedTarget 分页查询用户点赞记录（我的点赞页面）
// targetType=1帖子；targetType=2评论
func (r *LikeRepo) FindUserLikedTarget(ctx context.Context, userID int64, targetType int8, page int, pageSize int) ([]*domain.Like, int64, error) {
	var list []*domain.Like
	var total int64

	db := r.db.WithContext(ctx).Model(&domain.Like{}).
		Where("user_id = ? AND target_type = ?", userID, targetType)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&list).Error
	return list, total, err
}
