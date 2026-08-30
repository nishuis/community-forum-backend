package repository

import (
	"context"

	"github.com/nishuis/community-forum-backend/internal/domain"
	"gorm.io/gorm"
)

// PostRepo 持有gorm.DB实例，挂载数据库
type PostRepo struct {
	db *gorm.DB
}

// NewPostRepo 新建帖子仓库实例
func NewPostRepo(db *gorm.DB) *PostRepo {

	return &PostRepo{db: db}
}

// FindPostByPostId 通过帖子ID查找帖子
func (r *PostRepo) FindPostByPostId(ctx context.Context, postId int64) (*[]domain.Post, error) {

	var post []domain.Post
	err := r.db.WithContext(ctx).Where("post_id = ?", postId).Order("created_at DESC").Find(&post).Error

	return &post, err
}

// FindPostByPostTitle 通过帖子精确标题查找帖子
func (r *PostRepo) FindPostByPostTitle(ctx context.Context, postTitle string) (*domain.Post, error) {

	var post domain.Post
	err := r.db.WithContext(ctx).Where("title = ?", postTitle).First(&post).Error

	return &post, err
}

// FindPostByTitleLike 标题模糊查询
func (r *PostRepo) FindPostByTitleLike(ctx context.Context, postTitle string) (*[]domain.Post,error) {
	var list []*domain.Post
	var total int64
	db := r.db.WithContext(ctx).Model(&domain.Post{})
	if title != ""
}

// FindPostByAuthorId 通过作者ID(关联着userID)查找帖子
func (r *PostRepo) FindPostByAuthorId(ctx context.Context, authorId int64) (*[]domain.Post, error) {

	var post []domain.Post
	err := r.db.WithContext(ctx).Where("author_id = ?", authorId).Order("created_at DESC").Find(&post).Error

	return &post, err
}

// CreatePost
func (r *PostRepo) CreatePost(ctx context.Context, post *domain.Post) error {

	return r.db.WithContext(ctx).Create(post).Error
}
