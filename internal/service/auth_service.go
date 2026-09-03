// service/auth_service.go —— 鉴权业务层：校验用户名与密码、签发 access/refresh JWT 令牌。
package service

import (
	"context"
	"errors"

	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/repository"
	jwtutil "github.com/nishuis/community-forum-backend/pkg/jwt_util"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

//处理token相关业务

// AuthService 鉴权业务结构体，依赖
type AuthService struct {
	userRepo *repository.UserRepo
	cfg      *configs.Config
}

// NewAuthService 新建鉴权服务
func NewAuthService(ur *repository.UserRepo, cfg *configs.Config) *AuthService {
	return &AuthService{
		userRepo: ur,
		cfg:      cfg,
	}
}

// Login 登录业务
func (as *AuthService) Login(ctx context.Context, username string, password string) (accessToken string, refreshToken string, err error) {
	//1.校验用户名是否存在
	user, err := as.userRepo.FindUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", errs.ErrUserNotFound
		}
		return "", "", err
	}

	//2.校验密码是否正确
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", "", errs.ErrPasswordWrong
	}

	//3.校验通过，生成Token
	accessToken, err = jwtutil.GenerateAccessToken(user.UserId, username, as.cfg.Jwt.Secret, as.cfg.Jwt.AccessExpHour)
	if err != nil {
		return "", "", err
	}
	refreshToken, err = jwtutil.GenerateRefreshToken(user.UserId, username, as.cfg.Jwt.Secret, as.cfg.Jwt.RefreshExpDay)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}
