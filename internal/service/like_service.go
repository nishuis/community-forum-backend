// service/like_service.go —— 点赞业务层：点赞、取消、查询状态与列表，处理重复点赞等业务错误。
package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/nishuis/community-forum-backend/internal/cache"
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"gorm.io/gorm"
)

// LikeService 点赞业务结构体，依赖 LikeRepo/PostRepo/CommentRepo/Cache 注入。
type LikeService struct {
	likeRepo    *repository.LikeRepo
	postRepo    *repository.PostRepo
	commentRepo *repository.CommentRepo
	cache       *cache.Cache
}

// likeStatusTTL 点赞状态缓存 TTL
const likeStatusTTL = 10 * time.Minute

// likeCountTTL 点赞计数缓存 TTL
const likeCountTTL = 10 * time.Minute

// likeUserListTTL 我的点赞列表缓存 TTL
const likeUserListTTL = 5 * time.Minute

// likeUserListCache 我的点赞列表缓存组合结构：分页列表 + 总数
type likeUserListCache struct {
	List  []*domain.Like `json:"list"`
	Total int64          `json:"total"`
}

// likeStatusCacheKey 点赞状态缓存 key：cf:like:status:{userId}:{type}:{targetId}
func likeStatusCacheKey(userId int64, targetType int8, targetID int64) string {
	return "cf:like:status:" + strconv.FormatInt(userId, 10) + ":" + strconv.Itoa(int(targetType)) + ":" + strconv.FormatInt(targetID, 10)
}

// likeCountCacheKey 点赞计数缓存 key：cf:like:count:{type}:{targetId}
func likeCountCacheKey(targetType int8, targetID int64) string {
	return "cf:like:count:" + strconv.Itoa(int(targetType)) + ":" + strconv.FormatInt(targetID, 10)
}

// likeUserListCacheKey 我的点赞列表缓存 key：cf:like:user:{userId}:{type}:{page}:{size}
func likeUserListCacheKey(userId int64, targetType int8, page, pageSize int) string {
	return "cf:like:user:" + strconv.FormatInt(userId, 10) + ":" + strconv.Itoa(int(targetType)) + ":" + strconv.Itoa(page) + ":" + strconv.Itoa(pageSize)
}

// likeUserListCachePattern 我的点赞列表失效模式：cf:like:user:{userId}:{type}:*
func likeUserListCachePattern(userId int64, targetType int8) string {
	return "cf:like:user:" + strconv.FormatInt(userId, 10) + ":" + strconv.Itoa(int(targetType)) + ":*"
}

// NewLikeService 新建点赞业务实例，外部注入 repo 与 cache 依赖。
func NewLikeService(likeRepo *repository.LikeRepo, postRepo *repository.PostRepo, commentRepo *repository.CommentRepo, cache *cache.Cache) *LikeService {
	return &LikeService{
		likeRepo:    likeRepo,
		postRepo:    postRepo,
		commentRepo: commentRepo,
		cache:       cache,
	}
}

// DoLike 执行点赞
// userId:点赞用户; targetType:1帖子 2评论; targetID:目标id
func (s *LikeService) DoLike(ctx context.Context, userId int64, targetType int8, targetID int64) error {
	//1.校验targetType
	if targetType != domain.LikeTargetTypePost && targetType != domain.LikeTargetTypeComment {
		return errs.ErrLikeTargetType
	}

	//2.校验点赞目标真实存在；评论点赞时记录所属帖子ID，用于后续失效评论分页缓存
	var commentPostID int64
	switch targetType {
	case domain.LikeTargetTypePost:
		_, err := s.postRepo.FindPostById(ctx, targetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrLikeTargetNotFound
			}
			return err
		}
	case domain.LikeTargetTypeComment:
		comment, err := s.commentRepo.FindCommentByID(ctx, targetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrLikeTargetNotFound
			}
			return err
		}
		commentPostID = comment.PostID
	}

	//3.判断是否已点赞（预检查直查 DB，最终防重靠数据库唯一索引）
	isLiked, err := s.likeRepo.IsUserLike(ctx, userId, targetType, targetID)
	if err != nil {
		return err
	}
	if isLiked {
		return errs.ErrLikeAlready
	}

	//4.分发调用repo事务方法
	var repoErr error
	switch targetType {
	case domain.LikeTargetTypePost:
		repoErr = s.likeRepo.DoPostLikeTx(ctx, userId, targetID)
	case domain.LikeTargetTypeComment:
		repoErr = s.likeRepo.DoCommentLikeTx(ctx, userId, targetID)
	}

	if repoErr != nil {
		// 兜底：并发竞态，IsUserLike之后其他请求完成点赞，数据库报唯一索引冲突
		if errors.Is(repoErr, gorm.ErrDuplicatedKey) {
			return errs.ErrLikeAlready
		}
		return repoErr
	}

	//5.写后失效：点赞状态、计数、我的点赞列表；连带失效帖子详情/评论分页（LikeCount 字段已变）
	s.cache.Delete(ctx, likeStatusCacheKey(userId, targetType, targetID))
	s.cache.Delete(ctx, likeCountCacheKey(targetType, targetID))
	s.cache.DeletePattern(ctx, likeUserListCachePattern(userId, targetType))
	if targetType == domain.LikeTargetTypePost {
		s.cache.Delete(ctx, postCacheKey(targetID))
	} else {
		s.cache.DeletePattern(ctx, commentListCachePattern(commentPostID))
	}
	return nil
}

// CancelLike 取消点赞
func (s *LikeService) CancelLike(ctx context.Context, userId int64, targetType int8, targetID int64) error {
	//1.判断类型是否正确
	if targetType != domain.LikeTargetTypePost && targetType != domain.LikeTargetTypeComment {
		return errs.ErrLikeTargetType
	}

	//2.校验目标是否存在；评论取消点赞时记录所属帖子ID，用于后续失效评论分页缓存
	var commentPostID int64
	switch targetType {
	case domain.LikeTargetTypePost:
		_, err := s.postRepo.FindPostById(ctx, targetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrLikeTargetNotFound
			}
			return err
		}
	case domain.LikeTargetTypeComment:
		comment, err := s.commentRepo.FindCommentByID(ctx, targetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errs.ErrLikeTargetNotFound
			}
			return err
		}
		commentPostID = comment.PostID
	}

	//3.判断是否已点赞（预检查直查 DB）
	isLiked, err := s.likeRepo.IsUserLike(ctx, userId, targetType, targetID)
	if err != nil {
		return err
	}
	if !isLiked {
		return errs.ErrLikeNotExist
	}

	//分发调用repo取消事务方法
	var repoErr error
	switch targetType {
	case domain.LikeTargetTypePost:
		repoErr = s.likeRepo.CancelPostLikeTx(ctx, userId, targetID)
	case domain.LikeTargetTypeComment:
		repoErr = s.likeRepo.CancelCommentLikeTx(ctx, userId, targetID)
	}
	if repoErr != nil {
		return repoErr
	}

	//写后失效：点赞状态、计数、我的点赞列表；连带失效帖子详情/评论分页（LikeCount 字段已变）
	s.cache.Delete(ctx, likeStatusCacheKey(userId, targetType, targetID))
	s.cache.Delete(ctx, likeCountCacheKey(targetType, targetID))
	s.cache.DeletePattern(ctx, likeUserListCachePattern(userId, targetType))
	if targetType == domain.LikeTargetTypePost {
		s.cache.Delete(ctx, postCacheKey(targetID))
	} else {
		s.cache.DeletePattern(ctx, commentListCachePattern(commentPostID))
	}
	return nil
}

// IsUserLike 判断单个目标点赞状态（用于详情页）（Cache-Aside：先查缓存，未命中查 DB 回填）
func (s *LikeService) IsUserLike(ctx context.Context, userId int64, targetType int8, targetID int64) (bool, error) {
	if targetType != domain.LikeTargetTypePost && targetType != domain.LikeTargetTypeComment {
		return false, errs.ErrLikeTargetType
	}

	key := likeStatusCacheKey(userId, targetType, targetID)
	var cached bool
	found, _ := s.cache.GetJSON(ctx, key, &cached)
	if found {
		return cached, nil
	}

	liked, err := s.likeRepo.IsUserLike(ctx, userId, targetType, targetID)
	if err != nil {
		return false, err
	}
	s.cache.SetJSON(ctx, key, liked, likeStatusTTL)
	return liked, nil
}

// CountLike 获取目标点赞总数（Cache-Aside：先查缓存，未命中查 DB 回填）
func (s *LikeService) CountLike(ctx context.Context, targetType int8, targetID int64) (int64, error) {
	if targetType != domain.LikeTargetTypePost && targetType != domain.LikeTargetTypeComment {
		return 0, errs.ErrLikeTargetType
	}

	key := likeCountCacheKey(targetType, targetID)
	var cached int64
	found, _ := s.cache.GetJSON(ctx, key, &cached)
	if found {
		return cached, nil
	}

	total, err := s.likeRepo.CountLike(ctx, targetType, targetID)
	if err != nil {
		return 0, err
	}
	s.cache.SetJSON(ctx, key, total, likeCountTTL)
	return total, nil
}

// GetLikedIdMap 批量获取一批targetId中用户点赞的id集合，返回map用于列表页内存匹配，解决N+1
// targetIds 传入当前页面的一批帖子/评论id
// 返回 map[targetId]bool true代表已点赞
func (s *LikeService) GetLikedIdMap(ctx context.Context, userId int64, targetType int8, targetIds []int64) (map[int64]bool, error) {
	resMap := make(map[int64]bool)
	// 用户未登录 或者id切片为空，直接返回空map
	if userId == 0 || len(targetIds) == 0 {
		return resMap, nil
	}
	if targetType != domain.LikeTargetTypePost && targetType != domain.LikeTargetTypeComment {
		return nil, errs.ErrLikeTargetType
	}

	likedIds, err := s.likeRepo.BatchGetUserLikeIds(ctx, userId, targetType, targetIds)
	if err != nil {
		return nil, err
	}
	for _, id := range likedIds {
		resMap[id] = true
	}
	return resMap, nil
}

// GetUserLikedTarget 分页查询【我的点赞】页面（Cache-Aside：先查缓存，未命中查 DB 回填）
// 返回like关系列表、总条数
func (s *LikeService) GetUserLikedTarget(ctx context.Context, userId int64, targetType int8, page int, pageSize int) ([]*domain.Like, int64, error) {
	if targetType != domain.LikeTargetTypePost && targetType != domain.LikeTargetTypeComment {
		return nil, 0, errs.ErrLikeTargetType
	}
	//分页参数兜底
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	key := likeUserListCacheKey(userId, targetType, page, pageSize)
	var cached likeUserListCache
	found, _ := s.cache.GetJSON(ctx, key, &cached)
	if found {
		return cached.List, cached.Total, nil
	}

	list, total, err := s.likeRepo.FindUserLikedTarget(ctx, userId, targetType, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	s.cache.SetJSON(ctx, key, likeUserListCache{List: list, Total: total}, likeUserListTTL)
	return list, total, nil
}
