// service/post_service.go —— 帖子业务层：发帖、删帖、编辑、查询与关键词搜索，校验标题规则与作者权限。
package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/nishuis/community-forum-backend/configs"
	"github.com/nishuis/community-forum-backend/internal/cache"
	"github.com/nishuis/community-forum-backend/internal/domain"
	"github.com/nishuis/community-forum-backend/internal/errs"
	"github.com/nishuis/community-forum-backend/internal/repository"
	"gorm.io/gorm"
)

// PostService 帖子业务结构体，依赖PostRepo注入
type PostService struct {
	postRepo *repository.PostRepo
	userRepo *repository.UserRepo
	config   *configs.Config
	cache    *cache.Cache
}

// postDetailTTL 帖子详情缓存 TTL（基础值，实际写入会加 ±30% 随机抖动）
const postDetailTTL = 10 * time.Minute

// postListTTL 帖子列表类缓存 TTL（作者列表、标题查询、关键词搜索）
const postListTTL = 5 * time.Minute

// postSearchCache 关键词搜索缓存组合结构：分页列表 + 总数
type postSearchCache struct {
	List  []*domain.Post `json:"list"`
	Total int64          `json:"total"`
}

// postCacheKey 帖子详情缓存 key：cf:post:{postId}
func postCacheKey(postId int64) string {
	return "cf:post:" + strconv.FormatInt(postId, 10)
}

// postAuthorListCacheKey 作者帖子列表缓存 key：cf:user:{authorId}:posts
func postAuthorListCacheKey(authorId int64) string {
	return "cf:user:" + strconv.FormatInt(authorId, 10) + ":posts"
}

// postTitleCacheKey 精确标题查询缓存 key：cf:post:title:{title}
func postTitleCacheKey(title string) string {
	return "cf:post:title:" + title
}

// postSearchCacheKey 关键词搜索分页缓存 key：cf:post:search:{kw}:{page}:{size}
func postSearchCacheKey(kw string, page, pageSize int) string {
	return "cf:post:search:" + kw + ":" + strconv.Itoa(page) + ":" + strconv.Itoa(pageSize)
}

// postSearchCachePattern 关键词搜索缓存失效模式：cf:post:search:*
func postSearchCachePattern() string {
	return "cf:post:search:*"
}

// NewPostService 新建业务结构体
func NewPostService(postRepo *repository.PostRepo, userRepo *repository.UserRepo, cfg *configs.Config, cache *cache.Cache) *PostService {
	return &PostService{
		postRepo: postRepo,
		userRepo: userRepo,
		config:   cfg,
		cache:    cache,
	}
}

// CreatePost 发帖业务
func (s *PostService) CreatePost(ctx context.Context, title string, content string, authorId int64) (*domain.Post, error) {
	//检查标题长度
	if len(title) > 50 {
		return nil, errs.ErrPostTitleTooLong
	}
	if len(title) == 0 {
		return nil, errs.ErrPostTitleEmpty
	}
	//检查用户是否还存在
	_, err := s.userRepo.FindUserByUserId(ctx, authorId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrUserNotFound
		}
		return nil, err
	}

	//组装domain.Post
	newPost := &domain.Post{
		Title:    title,
		AuthorID: authorId,
		Content:  content,
	}

	//调用CreatePost
	err = s.postRepo.CreatePost(ctx, newPost)
	if err != nil {
		return nil, err
	}

	//写后失效：新帖影响作者列表、搜索结果、对应标题列表
	s.cache.Delete(ctx, postAuthorListCacheKey(authorId))
	s.cache.Delete(ctx, postTitleCacheKey(title))
	s.cache.DeletePattern(ctx, postSearchCachePattern())
	return newPost, nil
}

// DeletePost 删帖业务
func (s *PostService) DeletePost(ctx context.Context, userId int64, postId int64) error {
	//1.查询帖子存在
	post, err := s.postRepo.FindPostById(ctx, postId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrPostNotExist
		}
		return err
	}
	//2.验权
	if post.AuthorID != userId {
		return errs.ErrPostNotAuthor
	}

	//3.删除帖子
	err = s.postRepo.DeletePost(ctx, postId)
	if err != nil {
		return err
	}

	//4.写后失效：删除成功删除缓存，避免残留脏数据
	s.cache.Delete(ctx, postCacheKey(postId))
	//追加失效：作者列表、标题缓存、搜索结果、该帖评论分页缓存
	s.cache.Delete(ctx, postAuthorListCacheKey(post.AuthorID))
	s.cache.Delete(ctx, postTitleCacheKey(post.Title))
	s.cache.DeletePattern(ctx, postSearchCachePattern())
	s.cache.DeletePattern(ctx, commentListCachePattern(postId))
	return nil
}

// UpdatePost 编辑帖子
func (s *PostService) UpdatePost(ctx context.Context, userId int64, postId int64, updateTitle string, updateContent string) error {
	//1.验非空
	if updateTitle == "" && updateContent == "" {
		return errs.ErrParamWrong
	}

	//2..查询帖子存在
	post, err := s.postRepo.FindPostById(ctx, postId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrPostNotExist
		}
		return err
	}
	//3..验权
	if post.AuthorID != userId {
		return errs.ErrPostNotAuthor
	}

	//4.更新帖子
	rows, err := s.postRepo.UpdatePost(ctx, postId, updateTitle, updateContent)
	if err != nil {
		return err
	}
	if rows == 0 {
		return errs.ErrPostNotExist
	}

	//5.写后失效：更新成功删除缓存，下次读取时重建
	s.cache.Delete(ctx, postCacheKey(postId))
	//追加失效：作者列表、搜索结果；标题变更时同步失效旧/新标题缓存
	s.cache.Delete(ctx, postAuthorListCacheKey(post.AuthorID))
	s.cache.DeletePattern(ctx, postSearchCachePattern())
	if updateTitle != "" && updateTitle != post.Title {
		s.cache.Delete(ctx, postTitleCacheKey(post.Title))
		s.cache.Delete(ctx, postTitleCacheKey(updateTitle))
	}
	return nil
}

// GetPostById 获取单条帖子业务（Cache-Aside：先查缓存，未命中查 DB 回填）
func (s *PostService) GetPostById(ctx context.Context, postId int64) (*domain.Post, error) {
	key := postCacheKey(postId)

	// 1.先查缓存
	var cached domain.Post
	found, empty := s.cache.GetJSON(ctx, key, &cached)
	if found {
		return &cached, nil
	}
	if empty {
		// 命中空值占位：此前已确认帖子不存在（防穿透）
		return nil, errs.ErrPostNotExist
	}

	// 2.缓存未命中，查 DB
	post, err := s.postRepo.FindPostById(ctx, postId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 3a.防穿透：不存在的 ID 写空值占位（短 TTL 快速自愈）
			s.cache.SetEmpty(ctx, key)
			return nil, errs.ErrPostNotExist
		}
		return nil, err
	}

	// 3b.回填缓存
	s.cache.SetJSON(ctx, key, post, postDetailTTL)
	return post, nil
}

// GetPostByAuthorId 获取某用户全部帖子（Cache-Aside：先查缓存，未命中查 DB 回填）
func (s *PostService) GetPostByAuthorId(ctx context.Context, authorId int64) ([]*domain.Post, error) {
	// 校验用户是否存在
	_, err := s.userRepo.FindUserByUserId(ctx, authorId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrUserNotFound
		}
		return nil, err
	}

	key := postAuthorListCacheKey(authorId)

	// 1.先查缓存
	var cached []*domain.Post
	found, _ := s.cache.GetJSON(ctx, key, &cached)
	if found {
		return cached, nil
	}

	// 2.缓存未命中，查 DB
	list, err := s.postRepo.FindPostByAuthorId(ctx, authorId)
	if err != nil {
		return nil, err
	}

	// 3.回填缓存（空列表也缓存，避免重复查 DB）
	s.cache.SetJSON(ctx, key, list, postListTTL)
	return list, nil
}

// GetPostByTitleExact 精确标题查询帖子（Cache-Aside：先查缓存，未命中查 DB 回填）
func (s *PostService) GetPostByTitleExact(ctx context.Context, postTitle string) ([]*domain.Post, error) {
	postTitle = strings.TrimSpace(postTitle)
	if postTitle == "" {
		return nil, errs.ErrParamWrong
	}

	key := postTitleCacheKey(postTitle)

	// 1.先查缓存
	var cached []*domain.Post
	found, _ := s.cache.GetJSON(ctx, key, &cached)
	if found {
		return cached, nil
	}

	// 2.缓存未命中，查 DB
	list, err := s.postRepo.FindPostByTitle(ctx, postTitle)
	if err != nil {
		return nil, err
	}

	// 3.回填缓存（空列表也缓存）
	s.cache.SetJSON(ctx, key, list, postListTTL)
	return list, nil
}

// ShowByKeyWordOffset 关键词模糊搜索（Cache-Aside：先查缓存，未命中查 DB 回填）
func (s *PostService) ShowByKeyWordOffset(ctx context.Context, keyWord string, page int, pageSize int) ([]*domain.Post, int64, error) {
	//1.校正参数
	const maxPageSize = 50
	if pageSize <= 0 || pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	if page < 1 {
		page = 1
	}
	//剔除开头结尾空格
	keyWord = strings.TrimSpace(keyWord)
	// 关键词长度保护，防止超长搜索拖慢数据库
	if len(keyWord) > 50 {
		return nil, 0, errs.ErrParamWrong
	}

	key := postSearchCacheKey(keyWord, page, pageSize)

	//2.先查缓存
	var cached postSearchCache
	found, _ := s.cache.GetJSON(ctx, key, &cached)
	if found {
		return cached.List, cached.Total, nil
	}

	//3.缓存未命中，调用repo
	list, total, err := s.postRepo.ShowByKeyWordOffset(ctx, keyWord, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	//4.校验offset是否超限,即有没有查到有效数据
	offset := (page - 1) * pageSize
	if int64(offset) > total {
		empty := []*domain.Post{}
		s.cache.SetJSON(ctx, key, postSearchCache{List: empty, Total: total}, postListTTL)
		return empty, total, nil
	}

	//5.回填缓存
	s.cache.SetJSON(ctx, key, postSearchCache{List: list, Total: total}, postListTTL)
	return list, total, nil
}
