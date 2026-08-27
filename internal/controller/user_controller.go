package controller

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/internal/dto/request"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/service"
)

/*
接收HTTP/Gin请求，把JSON映射到dto/request入参结构体，做参数校验
调用service方法，本身不写业务逻辑，不接触数据库
将service返回的domain模型转换为dto/response输出结构体，返回JSON给前端，过滤敏感字段（Password）
*/

// UserController 用户控制器 持有service的引用
// service 实例从main函数注入进来，不在controller内部创建
// 便于单元测试，不用启动完整业务链
type UserController struct {
	userService *service.UserService
}

// NewUserController 新建用户控制器
func NewUserController(userService *service.UserService) *UserController {
	return &UserController{userService: userService}
}

// Register 用户注册接口
// POST
func (uc *UserController) Register(ctx *gin.Context) {
	//1.声明请求结构体，接收前端JSON，结构体来自dto/request
	var req request.RegisterReq
	//ShouldBindJSON 把http请求body的json绑定到req，同时执行binding标签的参数校验
	//入参必须是指针，因为要修改结构体，传值会操作副本，校验失败返回错误
	if err := ctx.ShouldBindJSON(&req); err != nil {
		//响应前端
		fmt.Println("Bind error:", err.Error())
		ctx.JSON(http.StatusOK, gin.H{
			//参数校验，缺少username/password字段类型错误
			"code": errs.CodeParamError,
			"msg":  "请求参数错误",
		})
		return
	}
	//校验密码规范
	if len(req.Password) < 6 {
		ctx.JSON(http.StatusOK, gin.H{
			"code": errs.CodeParamError,
			"msg":  "密码长度不能少于6",
		})
		return
	}

	//2.调用service层执行业务
	userDomain, err := uc.userService.Register(req.Username, req.Password, req.Email)
	if err != nil {
		//service返回业务错误，传给前端
		switch {
		case errors.Is(err, errs.ErrUsernameExisted):
			ctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeUserExists,
				"msg":  "用户已存在",
			})
		case errors.Is(err, errs.ErrEmailExisted):
			ctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeUserExists,
				"msg":  "邮箱已注册",
			})
		default:
			log.Printf("注册业务错误：%v", err)
			ctx.JSON(http.StatusOK, gin.H{
				"code": errs.CodeServerInternal,
				"msg":  "服务器内部错误",
			})
		}
		return
	}

	//3.domain 转换response DTO,屏蔽password敏感字段
	//domain.User包含加密Password,不能传给前端
	resp := response.RegisterResp{
		UserID:   userDomain.ID,
		Username: userDomain.Username,
	}

	//4.返回JSON，“注册成功”
	ctx.JSON(http.StatusOK, gin.H{
		"code": errs.CodeOk,
		"msg":  "注册成功",
		"data": resp,
	})

}
