package controller

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/internal/dto/request"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/service"
	"gorm.io/gorm"
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
func (c *PostController) CreatePost(ginctx *gin.Context) {
	//1.获取userId,title,content
	val, ok := ginctx.Get("userId")
	if !ok {
		log.Printf("jwt中间件异常放行，未获取到userId")
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		return
	}
	var req request.CreatePostReq
	if err := ginctx.ShouldBindJSON(&req); err != nil {
		log.Printf("bind error: %v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "请求参数错误",
		})
		return
	}

	//2.获取int64格式userId
	userId, ok := val.(int64)
	if !ok {
		log.Printf("jwt中间件异常放行，userId类型异常")
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		return
	}

	//3.调用service，透传原生context
	post, err := c.postService.CreatePost(ginctx.Request.Context(), req.Title, req.Content, userId)
	if err != nil {
		//处理context错误
		if errors.Is(err, context.Canceled) {
			log.Printf("发帖请求客户端主动取消: %v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeContextCancel,
				"msg":  "服务取消",
			})
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("请求超时：%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeDeadLineExceeded,
				"msg":  "超时未响应",
			})
			return
		}

		//处理业务错误
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeUserNotExist,
				"msg":  "用户不存在",
			})
			return
		}

		log.Printf("发帖业务错误：%v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		return
	}
	resp := response.CreatePostResp{
		PostId: post.PostId,
		Title:  post.Title,
	}

	//4.成功响应
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeCreated,
		"msg":  "创建成功",
		"data": resp,
	})

}
