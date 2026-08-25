package service

import (
	"errors"

	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService 用户业务结构体，依赖UserRepo注入
type UserService struct {
	userRepo *repository.UserRepo
}

// NewUserService 构造函数，外部注入repo依赖，不在内部new repo
func NewUserService(userRepo *repository.UserRepo) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) Register(username string, password string) (*domain.User, error) {
	//1.校验用户名是否存在
	_, err := s.userRepo.FindUserByUsername(username)
	if err != nil {
		//数据库查询出错，上抛到service层
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		//err=nil 查询成功且用户已存在
	} else {
		return nil, errors.New("用户名已存在")
	}
	//err not nil 且是ErrRecordNotFound，数据库查询成功但未找到用户，即用户名不存在，继续注册service

	//2.校验密码强度
	if len(password) < 6 {
		return nil, errors.New("密码长度不能少于6")
	}

	//3.加密明文密码
	hsahPwdBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	//4.组装domain.User模型
	newUser := &domain.User{Username: username,
		Password: string(hsahPwdBytes)}

	//5.调用repo，插入数据库，service不写gorm语句
	err = s.userRepo.CreateUser(newUser)
	if err != nil {
		return nil, err
	}

	//6.返回用户模型
	return newUser, nil
}
