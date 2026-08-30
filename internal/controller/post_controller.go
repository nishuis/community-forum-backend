package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/internal/service"
)

// PostController 帖子控制器
type PostController struct {
	postService *service.PostService
}

// NewPostController 新建控制器
func NewPostController(ps *service.PostService) *PostController {
	return &PostController{postService: ps}
}

// CreatePost 发帖接口
func (ps *PostController) CreatePost(ginctx *gin.Context)
