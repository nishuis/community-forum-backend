package repository

import (
	"context"
	"strings"

	"github.com/nishuis/community-forum-backend/internal/domain"
	"gorm.io/gorm"
)

// PostRepo 持有gorm.DB对象，挂载数据库
type PostRepo struct {
	db *gorm.DB
}

// NewPostRepo 新建帖子仓库实例
func NewPostRepo(db *gorm.DB) *PostRepo {

	return &PostRepo{db: db}
}

// Escape 转义工具
func Escape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// FindPostById 通过帖子ID查找帖子
func (r *PostRepo) FindPostById(ctx context.Context, postId int64) (*domain.Post, error) {

	var post domain.Post
	err := r.db.WithContext(ctx).Where("post_id = ?", postId).First(&post).Error

	return &post, err
}

// FindPostByTitle 通过帖子精确标题查找帖子
func (r *PostRepo) FindPostByTitle(ctx context.Context, postTitle string) ([]*domain.Post, error) {

	var posts []*domain.Post
	postTitle = Escape(postTitle)
	err := r.db.WithContext(ctx).Where("title = ?", postTitle).Find(&posts).Error

	return posts, err
}

// FindPostByTitleLike 标题模糊查询并分页
func (r *PostRepo) FindPostByTitleLike(ctx context.Context, postTitle string, page int, pageSize int) ([]*domain.Post, int64, error) {

	var list []*domain.Post
	var total int64

	// WithContext 返回绑定了ctx的新*gorm.DB实例，监听取消/超时信号
	// Model 指定操作 domain.Post, GORM自动映射posts表，自动追加deleted_at IS NULL(软删除过滤)
	db := r.db.WithContext(ctx).Model(&domain.Post{})
	postTitle = Escape(postTitle)
	db = db.Where("title LIKE ?", "%"+postTitle+"%")

	// Count 调用并执行拼好的Where条件，获取查到的个数
	// 执行完后db上的Where条件依然保留
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 链式叠加，追加排序，分页，查询
	// Find查不到会返回空切片
	err := db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error

	return list, total, err

}

// FindPostByAuthorId 通过作者ID(关联着userID)查找帖子
func (r *PostRepo) FindPostByAuthorId(ctx context.Context, authorId int64) ([]*domain.Post, error) {

	var posts []*domain.Post
	err := r.db.WithContext(ctx).Where("author_id = ?", authorId).Order("created_at DESC").Find(&posts).Error

	return posts, err
}

// CreatePost 新建帖子
func (r *PostRepo) CreatePost(ctx context.Context, post *domain.Post) error {

	return r.db.WithContext(ctx).Create(post).Error
}
