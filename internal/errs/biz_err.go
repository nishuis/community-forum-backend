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
)

// 错误码
const (
	CodeOk             = 200  //成功
	CodeParamError     = 1001 //请求参数错误
	CodeAuthFail       = 1002 //认证失败
	CodeServerInternal = 1003 //服务器内部错误
	CodeUserExists     = 1004 //用户名已存在
)
