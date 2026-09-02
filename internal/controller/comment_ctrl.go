package controller

import (
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

type CommentController struct {
	commentService *service.CommentService
}

func NewCommentController(svc *service.CommentService) *CommentController {
	return &CommentController{commentService: svc}
}

// CreateComment 发表评论 POST /api/comment/create 需要鉴权
func (c *CommentController) CreateComment(ginctx *gin.Context) {
	// 从jwt上下文获取userId
	val, ok := ginctx.Get("userId")
	if !ok {
		log.Printf("jwt中间件异常放行，未获取userId")
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

	var req request.CreateCommentReq
	if err := ginctx.ShouldBindJSON(&req); err != nil {
		log.Printf("CreateComment bind json err: %v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "请求参数错误",
		})
		return
	}

	comment, err := c.commentService.CreateComment(ginctx.Request.Context(), req.PostID, req.ParentCommentID, req.Content, userId)
	if err != nil {
		// 统一工具处理 context cancel / deadline
		if handled := ginutil.HandleContextError(ginctx, err, "CreateComment"); handled {
			return
		}

		switch {
		case errors.Is(err, errs.ErrCommentContentEmpty),
			errors.Is(err, errs.ErrCommentContentTooLong),
			errors.Is(err, errs.ErrPostNotExist),
			errors.Is(err, errs.ErrParentCommentNotExist),
			errors.Is(err, errs.ErrParentCommentNotBelongPost):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeParamError,
				"msg":  err.Error(),
			})
			return
		default:
			log.Printf("CreateComment service err:%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
			return
		}
	}

	resp := convert.ConvertCreateCommentResp(comment)
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeCreated,
		"msg":  "评论成功",
		"data": resp,
	})
}

// DeleteComment 删除评论 DELETE /api/comment/:comment_id 需要鉴权
func (c *CommentController) DeleteComment(ginctx *gin.Context) {
	val, ok := ginctx.Get("userId")
	if !ok {
		log.Printf("jwt中间件异常放行，未获取userId")
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

	commentIdStr := ginctx.Param("comment_id")
	commentId, err := strconv.ParseInt(commentIdStr, 10, 64)
	if err != nil {
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "评论ID参数错误",
		})
		return
	}

	err = c.commentService.DeleteComment(ginctx.Request.Context(), commentId, userId)
	if err != nil {
		if handled := ginutil.HandleContextError(ginctx, err, "DeleteComment"); handled {
			return
		}
		switch {
		case errors.Is(err, errs.ErrCommentNotFound),
			errors.Is(err, errs.ErrCommentNotAuthor):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeParamError,
				"msg":  err.Error(),
			})
			return
		default:
			log.Printf("DeleteComment service err:%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
			return
		}
	}

	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeDeleted,
		"msg":  "删除成功",
	})
}

// UpdateComment 编辑评论 POST /api/comment/update
func (c *CommentController) EUpdateComment(ginctx *gin.Context) {
	// 1.获取登录用户
	val, ok := ginctx.Get("userId")
	if !ok {
		log.Printf("jwt中间件异常放行，未获取userId")
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

	// 2.绑定参数
	var req request.UpdateCommentReq
	if err := ginctx.ShouldBindJSON(&req); err != nil {
		log.Printf("EditComment bind err: %v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "参数错误",
		})
		return
	}

	// 3.调用service
	comment, err := c.commentService.EditComment(ginctx.Request.Context(), req.CommentID, req.Content, userId)
	if err != nil {
		// 统一处理context超时/取消
		if handled := ginutil.HandleContextError(ginctx, err, "EditComment"); handled {
			return
		}

		switch {
		case errors.Is(err, errs.ErrCommentContentEmpty),
			errors.Is(err, errs.ErrCommentContentTooLong),
			errors.Is(err, errs.ErrCommentNotFound),
			errors.Is(err, errs.ErrCommentNotAuthor):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeNoContent,
				"msg":  err.Error(),
			})
			return
		default:
			log.Printf("EditComment service err: %v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
			return
		}
	}

	// 4.成功返回
	resp := convert.ConvertEditCommentResp(comment)
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOK,
		"msg":  "编辑成功",
		"data": resp,
	})
}

// GetCommentList 获取帖子评论列表 GET /api/post/:post_id/comments 公开接口无需鉴权
// query: page,size
func (c *CommentController) GetCommentList(ginctx *gin.Context) {
	postIdStr := ginctx.Param("post_id")
	postId, err := strconv.ParseInt(postIdStr, 10, 64)
	if err != nil {
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "帖子ID参数错误",
		})
		return
	}

	pageStr := ginctx.DefaultQuery("page", "1")
	pageSizeStr := ginctx.DefaultQuery("page_size", "20")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		pageSize = 20
	}

	list, total, err := c.commentService.GetCommentPageByPostId(ginctx.Request.Context(), postId, page, pageSize)
	if err != nil {
		if handled := ginutil.HandleContextError(ginctx, err, "GetCommentList"); handled {
			return
		}
		switch {
		case errors.Is(err, errs.ErrPostNotExist):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeNoContent,
				"msg":  err.Error(),
			})
			return
		default:
			log.Printf("GetCommentList service err:%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
			return
		}
	}

	// po转dto
	itemList := make([]*response.CommentItem, 0, len(list))
	for _, po := range list {
		itemList = append(itemList, convert.ConvertCommentItem(po))
	}

	totalPage := (total + int64(pageSize) - 1) / int64(pageSize)

	resp := response.OffsetPageResp[*response.CommentItem]{
		List:      itemList,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
	}

	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOK,
		"msg":  "success",
		"data": resp,
	})
}
