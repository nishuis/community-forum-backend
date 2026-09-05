package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	// 主键必须与库中现有 bigint unsigned 一致，否则外键关联（author_id 等）建表报类型不兼容
	UserId    int64          `gorm:"type:BIGINT UNSIGNED;primaryKey;autoIncrement;comment:'主键ID'"`
	CreatedAt time.Time      `gorm:"comment:'创建时间'"`
	UpdatedAt time.Time      `gorm:"comment:'更新时间'"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:'删除时间'"`

	Username string `gorm:"size:30;not null;uniqueIndex:uk_username;comment:'用户名，不能为空'"`
	Password string `gorm:"size:255;not null;comment:'密码，不存明文存哈希'" json:"-"`
	Email    string `gorm:"size:150;not null;uniqueIndex:uk_email;comment:'邮箱'"`
	Nickname string `gorm:"size:30;not null;default:'';comment:'昵称'"`
	Avatar   string `gorm:"size:255;not null;default:'';comment:'头像URL'"`
	Role     int8   `gorm:"type:TINYINT;not null;default:0;comment:'用户-0，管理员-1'"`
}

func (User) TableName() string {
	return "users"
}
