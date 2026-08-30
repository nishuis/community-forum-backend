package repository

import (
	"context"

	"github.com/nishuis/community-forum-backend/internal/domain"
	"gorm.io/gorm"
)

// UserRepo 结构体，持有gorm.DB实例，挂载数据库方法
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo 新建用户仓库实例
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

// FindUserByUsername 根据用户名查询用户
// *domain.User 查到的用户模型；error gorm产生的错误
func (r *UserRepo) FindUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User

	//err:1. gorm.ErrRecordNotFound: 没有找到这条记录（用户名不存在）
	//	  2.其他err： 数据库连接错误、sql语法错误等真正的异常
	//上抛错误到service，仓库层不处理
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	return &user, err

}

// FindUserByEmail 根据邮箱查找用户
func (r *UserRepo) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User

	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error

	return &user, err
}

// FindUserByUserId 根据用户ID查询用户
func (r *UserRepo) FindUserByUserId(ctx context.Context, userId int64) (*domain.User, error) {
	var user domain.User

	err := r.db.WithContext(ctx).Where("user_id = ?", userId).First(&user).Error

	return &user, err
}

// CreateUser 创建新用户，向user表中插入记录
// *domain.User 组装好的用户模型,在service层加密密码，repo不处理
// 返回：error 数据库插入时产生的错误（连接失败，唯一性冲突等）
func (r *UserRepo) CreateUser(ctx context.Context, user *domain.User) error {
	// db.Create: gorm 提供的插入方法，传入模型指针，自动提取结构体字段生成INSERT SQL
	err := r.db.WithContext(ctx).Create(user).Error
	return err
}
