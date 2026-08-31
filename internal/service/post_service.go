package service

import (
	"context"

	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/repository"
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
