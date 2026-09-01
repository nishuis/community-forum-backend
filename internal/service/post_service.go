package service

import (
	"context"
	"errors"
	"strings"

	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"gorm.io/gorm"
)

// PostService 帖子业务结构体，依赖PostRepo注入
type PostService struct {
	postRepo *repository.PostRepo
	userRepo *repository.UserRepo
	config   *configs.Config
}

// NewPostService 新建业务结构体
func NewPostService(postRepo *repository.PostRepo, userRepo *repository.UserRepo, cfg *configs.Config) *PostService {
	return &PostService{
		postRepo: postRepo,
		userRepo: userRepo,
		config:   cfg,
	}
}

// CreatePost 发帖业务
func (s *PostService) CreatePost(ctx context.Context, title string, content string, authorId int64) (*domain.Post, error) {
	//检查标题长度
	if len(title) > 50 {
		return nil, errs.ErrPostTitleTooLong
	}
	if len(title) == 0 {
		return nil, errs.ErrPostTitleEmpty
	}
	//检查用户是否还存在
	_, err := s.userRepo.FindUserByUserId(ctx, authorId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrUserNotFound
		}
		return nil, err
	}

	//组装domain.Post
	newPost := &domain.Post{
		Title:    title,
		AuthorID: authorId,
		Content:  content,
	}

	//调用CreatePost
	err = s.postRepo.CreatePost(ctx, newPost)

	return newPost, err
}

// DeletePost 删帖业务
func (s *PostService) DeletePost(ctx context.Context, userId int64, postId int64) error {
	//1.查询帖子存在
	post, err := s.postRepo.FindPostById(ctx, postId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrPostNotExist
		}
		return err
	}
	//2.验权
	if post.AuthorID != userId {
		return errs.ErrPostNotAuthor
	}

	//3.删除帖子
	err = s.postRepo.DeletePost(ctx, postId)
	return err
}

// UpdatePost 编辑帖子
func (s *PostService) UpdatePost(ctx context.Context, userId int64, postId int64, updateTitle string, updateContent string) error {
	//1.验非空
	if updateTitle == "" && updateContent == "" {
		return errs.ErrParamWrong
	}

	//2..查询帖子存在
	post, err := s.postRepo.FindPostById(ctx, postId)
	if err != nil {
		return errs.ErrPostNotExist
	}
	//3..验权
	if post.AuthorID != userId {
		return errs.ErrPostNotAuthor
	}

	//4.更新帖子
	rows, err := s.postRepo.UpdatePost(ctx, postId, updateTitle, updateContent)
	if err != nil {
		return err
	}
	if rows == 0 {
		return errs.ErrPostNotExist
	}
	return nil
}

// ShowByKeyWordOffset 关键词模糊搜索
func (s *PostService) ShowByKeyWordOffset(ctx context.Context, keyWord string, page int, pageSize int) ([]*domain.Post, int64, error) {
	//1.校正参数
	const maxPageSize = 50
	if pageSize <= 0 || pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	if page < 1 {
		page = 1
	}
	//剔除开头结尾空格
	keyWord = strings.TrimSpace(keyWord)

	//2.调用repo
	list, total, err := s.postRepo.ShowByKeyWordOffset(ctx, keyWord, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	//3.校验offset是否超限,即有没有查到有效数据
	offset := (page - 1) * pageSize
	if int64(offset) > total {
		return []*domain.Post{}, total, nil
	}
	return list, total, nil
}
