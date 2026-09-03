package controller

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/internal/dto/convert"
	"github.com/nishuis/community-forum-backend/internal/dto/request"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/service"
	ginutil "github.com/nishuis/community-forum-backend/pkg/gin_util"
)

// PostController 只管帖子资源 CRUD，不生产 JWT 令牌
type PostController struct {
	postService *service.PostService
}

// NewPostController 新建控制器
func NewPostController(ps *service.PostService) *PostController {
	return &PostController{postService: ps}
}

// CreatePost 发帖接口 POST请求
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
		if errors.Is(err, errs.ErrUserNotFound) {
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

// DeletePost 删帖端口 DELETE请求
func (c *PostController) DeletePost(ginctx *gin.Context) {
	//1.获取userId
	val, ok := ginctx.Get("userId")
	if !ok {
		log.Printf("jwt中间件异常放行，未获取到userId")
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		return
	}
	userId, ok := val.(int64)
	if !ok {
		log.Printf("jwt中间件异常放行，userId类型异常")
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		return
	}

	//2.从URL获取post_id
	postIdStr := ginctx.Param("post_id")
	postId, err := strconv.ParseInt(postIdStr, 10, 64)
	if err != nil {
		log.Printf("post_id 参数错误: %v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "post_id 参数格式错误",
		})
		return
	}

	//3.调用删帖服务
	err = c.postService.DeletePost(ginctx.Request.Context(), userId, postId)
	if err != nil {
		//处理context错误
		if errors.Is(err, context.Canceled) {
			log.Printf("删帖请求客户端主动取消: %v", err)
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
		switch {
		case errors.Is(err, errs.ErrPostNotAuthor):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeAuthFail,
				"msg":  "只能删除自己的帖子",
			})
		case errors.Is(err, errs.ErrPostNotExist):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodePostNotExist,
				"msg":  "内容不存在",
			})
		default:
			log.Printf("删帖业务错误：%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
		}
		return
	}

	//4.成功响应，无响应体
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeDeleted,
		"msg":  "删除成功",
	})
}

// UpdatePost 编辑帖子 PUT请求
func (c *PostController) UpdatePost(ginctx *gin.Context) {
	//1.获取当前用户基本信息
	userId, _, ok := ginutil.GetCurrentUserInfo(ginctx)
	if !ok {
		return
	}

	//2.从url获取postId,
	// 从请求体获取updateTitel,updateContent
	postIdStr := ginctx.Param("post_id")
	postId, err := strconv.ParseInt(postIdStr, 10, 64)
	if err != nil {
		log.Printf("post_id 参数错误: %v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "post_id 参数格式错误",
		})
		return
	}
	var req request.UpdatePostReq
	if err := ginctx.ShouldBindJSON(&req); err != nil {
		log.Printf("bind error: %v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "请求参数错误",
		})
		return
	}

	//3.调用服务
	err = c.postService.UpdatePost(ginctx.Request.Context(), userId, postId, req.UpdateTitle, req.UpdateContent)
	if err != nil {
		//处理context错误
		handle := ginutil.HandleContextError(ginctx, err, "编辑帖子业务")
		if handle {
			return
		}

		//处理业务错误
		switch {
		case errors.Is(err, errs.ErrPostNotAuthor):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeAuthFail,
				"msg":  "只能编辑自己的帖子",
			})
		case errors.Is(err, errs.ErrParamWrong):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeParamError,
				"msg":  "未修改任何内容",
			})
		case errors.Is(err, errs.ErrPostNotExist):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodePostNotExist,
				"msg":  "内容不存在",
			})
		default:
			log.Printf("更新帖子业务错误：%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
		}
		return
	}

	//4.成功响应，无响应体
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOK,
		"msg":  "编辑成功",
	})
}

// GetPostByKeyWordOffset 关键词查询帖子并分页 GET请求
func (c *PostController) GetPostByKeyWordOffset(ginctx *gin.Context) {
	//开放接口，无需鉴权

	//1.获取参数
	var req request.ShowByKeyWordOffsetReq
	err := ginctx.ShouldBindQuery(&req)
	if err != nil {
		log.Printf("ShowByKeyWordOffset绑定失败：%v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "参数错误",
		})
		return
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	//2.调用service
	ctx := ginctx.Request.Context()
	posts, total, err := c.postService.ShowByKeyWordOffset(ctx, req.KeyWord, req.Page, req.PageSize)
	if err != nil {
		//处理context错误
		handle := ginutil.HandleContextError(ginctx, err, "ShowByKeyWordOffset")
		if handle {
			return
		}

		//其他错误
		log.Printf("ShowByKeyWordOffset Error: %v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeServerInternal,
			"msg":  "服务器内部错误",
		})
		return
	}

	//绑定响应体，成功响应
	//计算总页数
	totalPage := (total + int64(req.PageSize) - 1) / int64(req.PageSize)
	//[]*domain.Post 转 []*response.PostListItem
	postList := convert.ConvertPostItemList(posts)
	//绑定通用分页返回结构体
	resp := response.OffsetPageResp[*response.PostItemResp]{
		List:      postList,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
		TotalPage: totalPage,
	}

	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOK,
		"msg":  "查询成功",
		"data": resp,
	})

}

// GetPostByPostId 获取帖子详情 GET 公开接口
func (c *PostController) GetPostByPostId(ginctx *gin.Context) {
	//1.获取路径参数post_id
	postIdStr := ginctx.Param("post_id")
	postId, err := strconv.ParseInt(postIdStr, 10, 64)
	if err != nil {
		log.Printf("post_id 参数错误: %v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "post_id 参数格式错误",
		})
		return
	}

	//2.调用service
	ctx := ginctx.Request.Context()
	post, err := c.postService.GetPostById(ctx, postId)
	if err != nil {
		//处理context取消/超时
		handle := ginutil.HandleContextError(ginctx, err, "GetPostDetail")
		if handle {
			return
		}
		switch {
		case errors.Is(err, errs.ErrPostNotExist):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodePostNotExist,
				"msg":  "内容不存在",
			})
		default:
			log.Printf("GetPostDetail 业务错误：%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
		}
		return
	}

	//3.domain转response
	respData := convert.ConvertPostItem(post)
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOK,
		"msg":  "查询成功",
		"data": respData,
	})
}

// GetAuthorPostList 获取某个用户发布的帖子列表 GET 公开接口
func (c *PostController) GetAuthorPostList(ginctx *gin.Context) {
	//1.路径参数 author_id
	authorIdStr := ginctx.Param("author_id")
	authorId, err := strconv.ParseInt(authorIdStr, 10, 64)
	if err != nil {
		log.Printf("author_id 参数错误: %v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "author_id 参数格式错误",
		})
		return
	}

	ctx := ginctx.Request.Context()
	postList, err := c.postService.GetPostByAuthorId(ctx, authorId)
	if err != nil {
		handle := ginutil.HandleContextError(ginctx, err, "GetAuthorPostList")
		if handle {
			return
		}
		switch {
		case errors.Is(err, errs.ErrUserNotFound):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeUserNotExist,
				"msg":  "用户不存在",
			})
		default:
			log.Printf("GetAuthorPostList 业务错误：%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
		}
		return
	}

	//domain 转响应结构体
	listResp := convert.ConvertPostItemList(postList)
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOK,
		"msg":  "查询成功",
		"data": listResp,
	})
}
