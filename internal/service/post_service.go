package service

import (
	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/repository"
)

// PostService 帖子业务结构体，依赖PostRepo注入
type PostService struct {
	postRepo *repository.PostRepo
	config   *configs.Config
}

// NewPostService 新建业务结构体
func NewPostService(postRepo *repository.PostRepo, cfg *configs.Config) *PostService {
	return &PostService{postRepo: postRepo, config: cfg}
}
