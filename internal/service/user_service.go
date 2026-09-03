// service/user_service.go —— 用户业务层：注册、查询、更新、注销用户，维护业务规则与业务错误映射。
package service

import (
	"context"
	"errors"
	"strings"

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
	//1.加密明文密码
	hsahPwdBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	//2.组装domain.User模型
	newUser := &domain.User{
		Username: username,
		Password: string(hsahPwdBytes),
		Email:    email,
	}

	//3.调用repo，插入数据库，service不写gorm语句
	err = s.userRepo.CreateUser(ctx, newUser)
	if err != nil {
		if s.userRepo.IsUniqueConstraintErr(err) {
			errMsg := err.Error()
			if strings.Contains(errMsg, "username") {
				return nil, errs.ErrUsernameExisted
			}
			if strings.Contains(errMsg, "email") {
				return nil, errs.ErrEmailExisted
			}
		}
		return nil, err
	}
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

// UpdateUserInfo 更新用户基础信息（不处理密码修改）
func (s *UserService) UpdateUserInfo(ctx context.Context, userId int64, username string, email string) (*domain.User, error) {
	// 查询原用户
	user, err := s.userRepo.FindUserByUserId(ctx, userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrUserNotFound
		}
		return nil, err
	}

	updateMap := make(map[string]interface{})

	// 如果传入新用户名，校验是否被别人占用
	if username != "" && username != user.Username {
		_, err := s.userRepo.FindUserByUsername(ctx, username)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
		} else {
			return nil, errs.ErrUsernameExisted
		}
		updateMap["username"] = username
	}
	// 如果传入新邮箱，校验是否被别人占用
	if email != "" && email != user.Email {
		_, err := s.userRepo.FindUserByEmail(ctx, email)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
		} else {
			return nil, errs.ErrEmailExisted
		}
		updateMap["email"] = email
	}

	if len(updateMap) == 0 {
		return user, nil
	}

	err = s.userRepo.UpdateUserByMap(ctx, userId, updateMap)
	if err != nil {
		if s.userRepo.IsUniqueConstraintErr(err) {
			errMsg := err.Error()
			if strings.Contains(errMsg, "username") {
				return nil, errs.ErrUsernameExisted
			}
			if strings.Contains(errMsg, "email") {
				return nil, errs.ErrEmailExisted
			}
		}
		return nil, err
	}

	updatedUser, err := s.userRepo.FindUserByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}
	return updatedUser, nil
}

// DeleteUser 删除自己账号
func (s *UserService) DeleteUser(ctx context.Context, userId int64) error {
	_, err := s.userRepo.FindUserByUserId(ctx, userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrUserNotFound
		}
		return err
	}
	return s.userRepo.DeleteUser(ctx, userId)
}
