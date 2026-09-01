package errs

import (
	"errors"
)

// 业务错误
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrPasswordWrong   = errors.New("password wrong")
	ErrUsernameExisted = errors.New("username already existed")
	ErrEmailExisted    = errors.New("email already existed")
	ErrParamWrong      = errors.New("params wrong")

	//Post错误
	ErrPostTitleTooLong = errors.New("post's title too long")
	ErrPostTitleEmpty   = errors.New("post's title can not be empty")
	ErrPostIdZero       = errors.New("postId can not be 0")
	ErrPostNotAuthor    = errors.New("not author")
	ErrPostNotExist     = errors.New("post not exist")
)

// 响应码
const (
	//成功响应
	CodeOK        = 200 //成功
	CodeCreated   = 201 //资源创建成功
	CodeDeleted   = 202 //资源删除成功
	CodeNoContent = 204 //成功但无响应体

	//客户端错误
	CodeParamError   = 1001 //请求参数错误
	CodeAuthFail     = 1002 //认证失败
	CodeUnauthorized = 1003 //未认证（缺少token）
	CodeUserExists   = 1004 //用户名已存在
	CodeUserNotExist = 1005 //用户不存在
	CodePostNotExist = 1006 //帖子不存在

	//Post错误

	//服务端错误
	CodeServerInternal   = 2001 //服务器内部错误
	CodeDeadLineExceeded = 2002 //context超时
	CodeContextCancel    = 2003 //context取消

)
