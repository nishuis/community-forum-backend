package errs

import (
	"errors"
)

// 业务错误
var (
	ErrUserNotFound    = errors.New("用户不存在")
	ErrPasswordWrong   = errors.New("密码错误")
	ErrUsernameExisted = errors.New("用户名已存在")
	ErrEmailExisted    = errors.New("邮箱已存在")
	ErrParamWrong      = errors.New("参数错误")

	//Post错误
	ErrPostTitleTooLong = errors.New("帖子标题过长")
	ErrPostTitleEmpty   = errors.New("帖子标题不能为空")
	ErrPostIdZero       = errors.New("帖子ID不能为0")
	ErrPostNotAuthor    = errors.New("不是帖子作者，无权操作")
	ErrPostNotExist     = errors.New("帖子不存在")

	//Comment错误
	ErrCommentContentEmpty        = errors.New("评论内容不能为空")
	ErrCommentContentTooLong      = errors.New("评论内容过长")
	ErrParentCommentNotExist      = errors.New("父评论不存在")
	ErrParentCommentNotBelongPost = errors.New("父评论不属于该帖子")
	ErrCommentNotFound            = errors.New("评论不存在")
	ErrCommentNotAuthor           = errors.New("不是评论作者，无权操作")

	//Like错误
	ErrLikeAlready        = errors.New("已经点赞，无需重复操作")
	ErrLikeNotExist       = errors.New("尚未点赞，无法取消")
	ErrLikeTargetType     = errors.New("点赞对象类型错误")
	ErrLikeTargetNotFound = errors.New("点赞目标不存在")
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
