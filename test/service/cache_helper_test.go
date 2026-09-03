// test/service —— 缓存测试公共辅助：miniredis（内存 Redis）+ go-redis 客户端 + cache 实例。
package service_test

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/nishuis/community-forum-backend/internal/cache"
	goredis "github.com/redis/go-redis/v9"
)

// newMiniredisCache 创建基于 miniredis 的内存 Redis，返回 server 与 cache 实例。
// server 可用于断言 key 是否存在、读取原始缓存值等；测试结束自动关闭。
func newMiniredisCache(t *testing.T) (*miniredis.Miniredis, *cache.Cache) {
	t.Helper()
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() 失败: %v", err)
	}
	t.Cleanup(srv.Close)

	client := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return srv, cache.NewCache(client)
}
