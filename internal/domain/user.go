package domain

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `gorm:"size:30;not null;uniqueIndex:uk_username;comment:'用户名，不能为空'"`
	Password string `gorm:"size:255;not null;comment:'密码，不存明文存哈希'"`
	Email    string `gorm:"size:150;not null;uniqueIndex:uk_email;comment:'邮箱'"`
	Nickname string `gorm:"sizw:30;not null;default:'';comment:'昵称'"`
	Avatar   string `gorm:"size:255;not null;default:'';comment:'头像URL'"`
	Role     int8   `gorm:"type:TINYINT;not null;default:0;comment:'用户-0，管理员-1'"`
}

func (User) TableName() string {
	return "users"
}
