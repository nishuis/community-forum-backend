// service/user_service.go —— 用户业务层：注册、查询、更新、注销用户，维护业务规则与业务错误映射。
package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/cache"
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
	cache    *cache.Cache
}

// CachedUser 用户缓存 DTO：剔除 Password，避免 bcrypt 哈希落入 Redis。
// 仅用于"读展示"，不得用于登录/密码校验（登录走 AuthService 直查 DB）。
type CachedUser struct {
	UserId    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Role      int8      `json:"role"`
}

// toCachedUser domain.User → CachedUser（丢弃 Password）
func toCachedUser(u *domain.User) *CachedUser {
	if u == nil {
		return nil
	}
	return &CachedUser{
		UserId:    u.UserId,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		Username:  u.Username,
		Email:     u.Email,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Role:      u.Role,
	}
}

// toDomain CachedUser → domain.User（Password 为空）
func (cu *CachedUser) toDomain() *domain.User {
	if cu == nil {
		return nil
	}
	return &domain.User{
		UserId:    cu.UserId,
		CreatedAt: cu.CreatedAt,
		UpdatedAt: cu.UpdatedAt,
		Username:  cu.Username,
		Email:     cu.Email,
		Nickname:  cu.Nickname,
		Avatar:    cu.Avatar,
		Role:      cu.Role,
	}
}

// userCacheTTL 用户信息缓存 TTL（基础值，实际写入会加 ±30% 随机抖动）
const userCacheTTL = 30 * time.Minute

// userCacheKey 用户信息缓存 key：cf:user:{userId}
func userCacheKey(userId int64) string {
	return "cf:user:" + strconv.FormatInt(userId, 10)
}

// NewUserService 构造函数，外部注入repo依赖，不在内部new repo
func NewUserService(userRepo *repository.UserRepo, conf *configs.Config, cache *cache.Cache) *UserService {
	return &UserService{userRepo: userRepo, conf: conf, cache: cache}
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

// GetCurrentUser 带token访问当前用户信息（Cache-Aside：先查缓存，未命中查 DB 回填）
func (s *UserService) GetCurrentUser(ctx context.Context, userId int64) (*domain.User, error) {
	key := userCacheKey(userId)

	// 1.先查缓存
	var cached CachedUser
	found, empty := s.cache.GetJSON(ctx, key, &cached)
	if found {
		return cached.toDomain(), nil
	}
	if empty {
		// 命中空值占位：此前已确认用户不存在（防穿透）
		return nil, errs.ErrUserNotFound
	}

	// 2.缓存未命中，查 DB
	user, err := s.userRepo.FindUserByUserId(ctx, userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 3a.防穿透：不存在的 ID 写空值占位
			s.cache.SetEmpty(ctx, key)
			return nil, errs.ErrUserNotFound
		}
		return nil, err
	}

	// 3b.回填缓存（剔除 Password，只存展示字段）
	s.cache.SetJSON(ctx, key, toCachedUser(user), userCacheTTL)
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

	// 更新成功后失效缓存，下次读取时重建
	s.cache.Delete(ctx, userCacheKey(userId))

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
	if err := s.userRepo.DeleteUser(ctx, userId); err != nil {
		return err
	}

	// 删除成功后失效缓存，避免残留脏数据
	s.cache.Delete(ctx, userCacheKey(userId))
	return nil
}
