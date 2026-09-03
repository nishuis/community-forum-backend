package controller

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/dto/convert"
	"github.com/nishuis/community-forum-backend/internal/dto/request"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/service"
	ginutil "github.com/nishuis/community-forum-backend/pkg/gin_util"
)

type LikeController struct {
	likeService *service.LikeService
	cfg         *configs.Config
}

func NewLikeController(svc *service.LikeService, cfg *configs.Config) *LikeController {
	return &LikeController{
		likeService: svc,
		cfg:         cfg,
	}
}

// DoLike 执行点赞 POST /api/like/do 需要鉴权
func (c *LikeController) DoLike(ginctx *gin.Context) {
	//1.获取当前操作用户信息
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

	//2.获取请求信息，targetType,TargetId
	var req request.DoLikeReq
	if err := ginctx.ShouldBindJSON(&req); err != nil {
		log.Printf("DoLike bind json err: %v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "请求参数错误",
		})
		return
	}

	//3.调用服务，处理错误
	err := c.likeService.DoLike(ginctx.Request.Context(), userId, req.TargetType, req.TargetID)
	if err != nil {
		//处理context错误
		if handled := ginutil.HandleContextError(ginctx, err, "DoLike"); handled {
			return
		}

		//处理业务错误
		switch {
		case errors.Is(err, errs.ErrLikeTargetType),
			errors.Is(err, errs.ErrLikeAlready),
			errors.Is(err, errs.ErrLikeTargetNotFound):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeAuthFail,
				"msg":  err.Error(),
			})
			return
		default:
			log.Printf("DoLike service err:%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
			return
		}
	}

	//4.成功响应
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOK,
		"msg":  "点赞成功",
	})
}

// CancelLike 取消点赞 POST /api/like/cancel 需要鉴权
func (c *LikeController) CancelLike(ginctx *gin.Context) {
	//1.获取当前操作用户信息
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

	//2.获取请求信息，targetType、targetId
	var req request.CancelLikeReq
	if err := ginctx.ShouldBindJSON(&req); err != nil {
		log.Printf("CancelLike bind json err: %v", err)
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "请求参数错误",
		})
		return
	}

	//3.调用服务，处理错误
	err := c.likeService.CancelLike(ginctx.Request.Context(), userId, req.TargetType, req.TargetID)
	if err != nil {
		//处理context错误
		if handled := ginutil.HandleContextError(ginctx, err, "CancelLike"); handled {
			return
		}

		//处理业务错误
		switch {
		case errors.Is(err, errs.ErrLikeTargetType),
			errors.Is(err, errs.ErrLikeNotExist),
			errors.Is(err, errs.ErrLikeTargetNotFound):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeNoContent,
				"msg":  err.Error(),
			})
			return
		default:
			log.Printf("CancelLike service err:%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
			return
		}
	}

	//4.响应成功
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOK,
		"msg":  "取消点赞成功",
	})
}

// GetMyLiked 获取我的点赞记录 GET /api/like/my 需要鉴权
// query target_type=1 page=1 page_size=20
func (c *LikeController) GetMyLiked(ginctx *gin.Context) {
	//1.获取当前用户信息
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

	//2.从URL中获取信息并校验类型
	//如果请求 url 中没有传对应 query 参数或者类型错误，就使用第二个参数作为默认返回
	targetTypeStr := ginctx.Query("target_type")
	pageStr := ginctx.DefaultQuery("page", "1")
	sizeStr := ginctx.DefaultQuery("page_size", "20")

	targetType, err := strconv.ParseInt(targetTypeStr, 10, 8)
	if err != nil {
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "target_type参数错误",
		})
		return
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(sizeStr)
	if err != nil || pageSize == 0 {
		pageSize = 20
	}

	//3.调用服务，处理错误
	list, total, err := c.likeService.GetUserLikedTarget(ginctx.Request.Context(), userId, int8(targetType), page, pageSize)
	if err != nil {
		//处理context错误
		if handled := ginutil.HandleContextError(ginctx, err, "GetMyLiked"); handled {
			return
		}

		//处理业务错误
		switch {
		case errors.Is(err, errs.ErrLikeTargetType):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeParamError,
				"msg":  err.Error(),
			})
			return
		default:
			log.Printf("GetMyLiked service err:%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
			return
		}
	}

	//4.把实体列表转换成响应单元列表，绑定返回体
	itemList := convert.ConvertUserLikedItemList(list)
	totalPage := (total + int64(pageSize) - 1) / int64(pageSize)

	resp := response.OffsetPageResp[*response.UserLikedItemResp]{
		List:      itemList,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
	}

	//5.响应成功
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOK,
		"msg":  "success",
		"data": resp,
	})
}

// GetLikeStatus 获取点赞状态 GET /api/like/status 可选鉴权
// 默认返回未点赞和点赞数，鉴权成功，返回用户点赞状态和点赞数
// query: target_type=1&target_id=10
func (c *LikeController) GetLikeStatus(ginctx *gin.Context) {
	//1.尝试鉴权，获取操作用户信息
	userId := ginutil.TryGetUserIdFromHeader(ginctx, c.cfg)

	//2.从URL中获取信息，检验参数类型
	targetTypeStr := ginctx.Query("target_type")
	targetIDStr := ginctx.Query("target_id")

	targetType, err := strconv.ParseInt(targetTypeStr, 10, 8)
	if err != nil {
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "target_type参数错误",
		})
		return
	}
	targetId, err := strconv.ParseInt(targetIDStr, 10, 64)
	if err != nil || targetId < 1 {
		ginctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "target_id参数错误",
		})
		return
	}

	//3.鉴权成功userId不为0，调用服务，判断点赞状态
	isLike := false
	if userId != 0 {
		isLike, err = c.likeService.IsUserLike(ginctx.Request.Context(), userId, int8(targetType), targetId)
		if err != nil {
			//处理context错误
			if handled := ginutil.HandleContextError(ginctx, err, "GetLikeStatus"); handled {
				return
			}
			//处理业务错误
			log.Printf("GetLikeStatus isLike err:%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
			return
		}
	}

	//4.获取点赞数
	count, err := c.likeService.CountLike(ginctx.Request.Context(), int8(targetType), targetId)
	if err != nil {
		if handled := ginutil.HandleContextError(ginctx, err, "GetLikeStatus"); handled {
			return
		}
		switch {
		case errors.Is(err, errs.ErrLikeTargetType):
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeParamError,
				"msg":  err.Error(),
			})
			return
		default:
			log.Printf("GetLikeStatus count err:%v", err)
			ginctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
			return
		}
	}

	//5.绑定响应体，响应成功
	resp := response.LikeStatusResp{
		IsLike: isLike,
		Count:  count,
	}
	ginctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOK,
		"msg":  "success",
		"data": resp,
	})
}
