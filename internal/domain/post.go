package domain

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID        int64          `gorm:"primaryKey;autoIncrement;comment:'主键ID'"`
	CreatedAt time.Time      `gorm:"comment:'创建时间'"`
	UpdatedAt time.Time      `gorm:"comment:'更新时间'"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:'删除时间'"`

	AuthorID int64 `gorm:"type:BIGINT UNSIGNED;not null;index;comment:'作者ID,关联user.ID'"`
	Author   User  `gorm:"foreignKey:AuthorID"`
	//CategoryID uint     `gorm:"type:BIGINT UNSIGNED;index:idx_posts_category_id;comment:'板块类型'"`
	//Category   Category `gorm:"foreignKey:CategoryID"`

	Title   string `gorm:"size:50;not null;index:idx_posts_title;comment:'标题'"`
	Summary string `gorm:"size:255;comment:'摘要'"`
	Content string `gorm:"type:TEXT;not null;comment:'正文'"`

	CommentCount    uint `gorm:"type:INT UNSIGNED;default:0;not null;comment:'评论数'"`
	LikeCount       uint `gorm:"type:INT UNSIGNED;default:0;not null;comment:'点赞数'"`
	CollectionCount uint `gorm:"type:INT UNSIGNED;default:0;not null;comment:'收藏数'"`
	ViewCount       uint `gorm:"type:INT UNSIGNED;default:0;not null;comment:'观看次数'"`

	IsTop    int8       `gorm:"type:TINYINT;not null;default:0;index:idx_is_top;comment:'是否置顶，0-未置顶，1-置顶'"`
	TopEndAt *time.Time `gorm:"index;comment:'置顶过期时间'"`
	Status   int8       `gorm:"type:TINYINT;not null;default:0;index:idx_status;comment:'状态，0-正常，1-审核中，2-屏蔽'"`
	//RejectReason string `gorm:"size:200;comment:'屏蔽/拒绝原因'"`

	LastCommentAt *time.Time `gorm:"index;comment:'最后回复时间'"`
}

func (Post) TableName() string {
	return "posts"
}

/*
id:主键，自增，无符号，bigint
createdat:
updatedat:
deletedat:
userid：关联user-id 作者
title：string not null Index 标题
content：string not null 内容
likes：bigint default 0 点赞数
comments：int default 0 评论数
collections: bigint default 0 收藏数
shares: bigint default 0 分享数
`view_count int default 0`：浏览次数
`is_top int8 default 0`：是否置顶 0 = 否 1 = 是
`status int8 default 0`：帖子状态：0 正常，1 审核中，2 封禁屏蔽
`category_id uint`：板块 ID
*/
