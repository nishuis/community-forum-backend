// service/comment_service.go —— 评论业务层：发表、删除、编辑评论与分页查询，校验内容长度与作者权限。
package service

import (
	"context"
	"errors"

	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"gorm.io/gorm"
)

// CommentService 评论业务结构体，依赖 CommentRepo/PostRepo 注入。
type CommentService struct {
	commentRepo *repository.CommentRepo
	postRepo    *repository.PostRepo
}

// NewCommentService 新建评论业务实例，外部注入 repo 依赖。
func NewCommentService(commentRepo *repository.CommentRepo, postRepo *repository.PostRepo) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		postRepo:    postRepo,
	}
}

// CreateComment 创建评论
// postId:归属帖子; parentCommentId:父评论id，0=一级评论; content:评论内容; authorId:评论作者
func (s *CommentService) CreateComment(ctx context.Context, postId int64, parentCommentId int64, content string, authorId int64) (*domain.Comment, error) {
	// 1.业务参数校验
	const contentMaxLen = 500
	if len(content) == 0 {
		return nil, errs.ErrCommentContentEmpty
	}
	if len(content) > contentMaxLen {
		return nil, errs.ErrCommentContentTooLong
	}

	// 2.校验帖子是否存在
	_, err := s.postRepo.FindPostById(ctx, postId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrPostNotExist
		}
		return nil, err
	}

	//3.如果是回复楼中楼，校验父评论
	if parentCommentId > 0 {
		parentComment, err := s.commentRepo.FindCommentByID(ctx, parentCommentId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errs.ErrParentCommentNotExist
			}
			return nil, err
		}
		// 防越权：父评论必须属于同一个帖子，不能跨帖子回复
		if parentComment.PostID != postId {
			return nil, errs.ErrParentCommentNotBelongPost
		}
	}

	//4.组装评论实体
	newComment := &domain.Comment{
		PostID:          postId,
		ParentCommentID: parentCommentId,
		AuthorID:        authorId,
		Content:         content,
		IsTop:           0,
		Status:          0,
	}

	err = s.commentRepo.CreateComment(ctx, newComment)
	if err != nil {
		return nil, err
	}
	return newComment, nil
}

// DeleteComment 删除评论，鉴权：只有评论作者本人可以删除
func (s *CommentService) DeleteComment(ctx context.Context, commentId int64, userId int64) error {
	//查询评论
	comment, err := s.commentRepo.FindCommentByID(ctx, commentId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrCommentNotFound
		}
		return err
	}
	//权限校验：是否评论作者
	if comment.AuthorID != userId {
		return errs.ErrCommentNotAuthor
	}

	err = s.commentRepo.DeleteComment(ctx, commentId)
	return err
}

// EditComment 编辑评论
// 权限：仅作者本人可编辑
func (s *CommentService) EditComment(ctx context.Context, commentId int64, content string, userId int64) (*domain.Comment, error) {
	// 1.参数校验
	const contentMaxLen = 500
	if len(content) == 0 {
		return nil, errs.ErrCommentContentEmpty
	}
	if len(content) > contentMaxLen {
		return nil, errs.ErrCommentContentTooLong
	}

	// 2.查询评论是否存在
	comment, err := s.commentRepo.FindCommentByID(ctx, commentId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrCommentNotFound
		}
		return nil, err
	}

	// 3.权限校验：必须是作者
	if comment.AuthorID != userId {
		return nil, errs.ErrCommentNotAuthor
	}

	// 4.更新数据库
	err = s.commentRepo.UpdateCommentContent(ctx, commentId, content)
	if err != nil {
		return nil, err
	}

	// 5.重新查询最新数据返回
	newComment, err := s.commentRepo.FindCommentByID(ctx, commentId)
	return newComment, err
}

// GetCommentPageByPostId 获取帖子分页评论列表
func (s *CommentService) GetCommentPageByPostId(ctx context.Context, postId int64, page int, pageSize int) ([]*domain.Comment, int64, error) {
	//简单校验page，兜底
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	//校验帖子是否存在
	_, err := s.postRepo.FindPostById(ctx, postId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errs.ErrPostNotExist
		}
		return nil, 0, err
	}

	list, total, err := s.commentRepo.FindCommentPageByPostID(ctx, postId, page, pageSize)
	return list, total, err
}
