package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nishuis/community-forum-backend/internal/dto/request"
	"github.com/nishuis/community-forum-backend/internal/dto/response"
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
		ctx.JSON(http.StatusOK, gin.H{
			//参数校验，缺少username/password字段类型错误，返回400
			"code":  1001,
			"msg":   "请求参数错误",
			"error": err.Error(),
		})
		return
	}

	//2.调用service层执行业务
	userDomain, err := uc.userService.Register(req.Username, req.Password)
	if err != nil {
		//service返回业务错误，传给前端
		ctx.JSON(http.StatusOK, gin.H{
			"code": 1002,
			"msg":  err.Error(),
		})
		return
	}

	//3.domain 转换response DTO,屏蔽password敏感字段
	//domain.User包含加密Password,不能传给前端
	resp := response.UserResp{
		UserID:   userDomain.ID,
		Username: userDomain.Username,
	}

	//4.返回JSON，“注册成功”
	ctx.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "注册成功",
		"data": resp,
	})

}
