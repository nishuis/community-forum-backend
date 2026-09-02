package domain

import (
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	CommentID int64 `gorm:"type:BIGINT UNSIGNED;primaryKey;autoIncrement;comment:'主键ID'"`

	CreatedAt time.Time      `gorm:"comment:'创建时间'"`
	UpdatedAt time.Time      `gorm:"comment:'更新时间'"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:'删除时间'"`

	PostID          int64 `gorm:"type:BIGINT UNSIGNED;not null;index:idx_postid;comment:'归属帖子ID'"`
	ParentCommentID int64 `gorm:"type:BIGINT UNSIGNED;not null;default:0;index:idx_parent;comment:'父评论ID，0为一级评论'"`

	AuthorID int64 `gorm:"type:BIGINT UNSIGNED;not null;index;comment:'作者ID,关联user.ID'"`
	Author   *User `gorm:"foreignKey:AuthorID"`

	IsTop    int8       `gorm:"type:TINYINT;not null;default:0;index:idx_is_top;comment:'是否置顶，0-未置顶，1-置顶'"`
	TopEndAt *time.Time `gorm:"index;comment:'置顶过期时间'"`
	Status   int8       `gorm:"type:TINYINT;not null;default:0;index:idx_status;comment:'状态，0-正常，1-审核中，2-屏蔽'"`

	Content string `gorm:"type:text;not null;comment:'评论内容'"`

	//CommentCount    uint `gorm:"type:INT UNSIGNED;default:0;not null;comment:'评论数'"`
	//LikeCount       uint `gorm:"type:INT UNSIGNED;default:0;not null;comment:'点赞数'"`
	//CollectionCount uint `gorm:"type:INT UNSIGNED;default:0;not null;comment:'收藏数'"`
	//ViewCount       uint `gorm:"type:INT UNSIGNED;default:0;not null;comment:'观看次数'"`

}
