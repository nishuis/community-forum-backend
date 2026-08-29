package service

import (
	"context"
	"errors"

	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService 用户业务结构体，依赖UserRepo注入
type UserService struct {
	userRepo *repository.UserRepo
	conf     *configs.Config
}

// NewUserService 构造函数，外部注入repo依赖，不在内部new repo
func NewUserService(userRepo *repository.UserRepo, conf *configs.Config) *UserService {
	return &UserService{userRepo: userRepo, conf: conf}
}

// Register 注册业务
func (s *UserService) Register(ctx context.Context, username string, password string, email string) (*domain.User, error) {
	//1.校验用户名是否存在
	_, err := s.userRepo.FindUserByUsername(ctx, username)
	if err != nil {
		//非“未找到”错误：数据库、context异常，直接上抛
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		// err为nil说明查询正常，并且查到用户，用户名重复
		return nil, errs.ErrUsernameExisted
	}

	//2.校验邮箱是否存在
	_, err = s.userRepo.FindUserByEmail(ctx, email)
	if err != nil {
		//数据库查询或Context出错，上抛到controller层
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	} else {
		return nil, errs.ErrEmailExisted
	}
	//err not nil 且是ErrRecordNotFound，数据库查询成功但未找到用户，即用户名不存在，继续注册service

	//3.加密明文密码
	hsahPwdBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	//4.组装domain.User模型
	newUser := &domain.User{Username: username,
		Password: string(hsahPwdBytes),
		Email:    email,
	}

	//5.调用repo，插入数据库，service不写gorm语句
	err = s.userRepo.CreateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}

	//6.返回用户模型
	return newUser, nil
}

// GetCurrentUser 带token访问当前用户信息
func (s *UserService) GetCurrentUser(ctx context.Context, userId int64) (*domain.User, error) {
	//1.校验用户名是否存在
	user, err := s.userRepo.FindUserByUserId(ctx, userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrUserNotFound
		}
		return nil, err
	}

	//2.返回用户
	return user, nil
}
