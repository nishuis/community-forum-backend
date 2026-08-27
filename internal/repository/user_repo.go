package repository

import (
	"errors"

	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"gorm.io/gorm"
)

// UserRepo 结构体，持有gorm.DB实例，挂载数据库方法
type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

// FindUserByUsername 根据用户名查询用户
// *domain.User 查到的用户模型；error gorm产生的错误
func (ur *UserRepo) FindUserByUsername(username string) (*domain.User, error) {
	var user domain.User

	//err:1. gorm.ErrRecordNotFound: 没有找到这条记录（用户名不存在）
	//	  2.其他err： 数据库连接错误、sql语法错误等真正的异常
	//上抛错误到service，仓库层不处理
	err := ur.db.Where("username = ?", username).First(&user).Error

	return &user, err

}

// CreateUser 创建新用户，向user表中插入记录
// *domain.User 组装好的用户模型,在service层加密密码，repo不处理
// 返回：error 数据库插入时产生的错误（连接失败，唯一性冲突等）
func (ur *UserRepo) CreateUser(user *domain.User) error {
	// db.Create: gorm 提供的插入方法，传入模型指针，自动提取结构体字段生成INSERT SQL
	err := ur.db.Create(user).Error
	//捕获唯一索引冲突，用户名已经检查过，这里只能是邮箱冲突
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errs.ErrEmailExisted
		}
	}
	return err
}
