package domain

import (
	"time"

	"gorm.io/gorm"
)

// Like 点赞 帖子点赞、评论点赞
// **唯一复合索引 `uk_user_target(user_id,target_type,target_id)`**
// 数据库层面防止同一用户重复点赞；重复插入返回 `gorm.ErrDuplicatedKey`，service 捕获转业务错误。
type Like struct {
	LikeID     int64 `gorm:"type:BIGINT UNSIGNED;primaryKey;autoIncrement;comment:'点赞主键'"`
	UserID     int64 `gorm:"type:BIGINT UNSIGNED;not null;index:uk_user_target,unique:uk_user_target;comment:'点赞用户ID'"`
	TargetType int8  `gorm:"type:TINYINT;not null;index:uk_user_target,unique:uk_user_target;comment:'点赞对象类型：1‑帖子post，2‑评论comment'"`
	TargetID   int64 `gorm:"type:BIGINT UNSIGNED;not null;index:uk_user_target,unique:uk_user_target;comment:'目标ID，post_id或者comment_id'"`

	CreatedAt time.Time      `gorm:"comment:'点赞时间'"`
	UpdatedAt time.Time      `gorm:"comment:'更新时间'"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:'软删除，取消点赞标记删除'"`
}

const (
	LikeTargetTypePost    int8 = 1 // 帖子
	LikeTargetTypeComment int8 = 2 // 评论
)
