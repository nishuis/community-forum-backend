// test/cache —— cache 包黑盒单测（package cache_test）。
// 用 miniredis（内存 Redis）验证 GetJSON/SetJSON/Delete/DeletePattern/SetEmpty
// 基础行为，以及 nil client 时的 no-op 降级。
package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/nishuis/community-forum-backend/internal/cache"
	goredis "github.com/redis/go-redis/v9"
)

// newTestCache 创建基于 miniredis 的内存 Redis，返回 server 与 cache 实例。
func newTestCache(t *testing.T) (*miniredis.Miniredis, *cache.Cache) {
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

// TestSetGetJSON 写入后能命中并正确反序列化，且真实落盘带 TTL。
func TestSetGetJSON(t *testing.T) {
	srv, c := newTestCache(t)
	ctx := context.Background()

	type item struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	c.SetJSON(ctx, "k:1", item{ID: 1, Name: "alice"}, time.Minute)

	var got item
	found, empty := c.GetJSON(ctx, "k:1", &got)
	if !found || empty {
		t.Fatalf("期望 found=true empty=false，实际 found=%v empty=%v", found, empty)
	}
	if got.ID != 1 || got.Name != "alice" {
		t.Fatalf("反序列化结果错误: %+v", got)
	}
	if !srv.Exists("k:1") {
		t.Fatal("key 未写入 miniredis")
	}
	if srv.TTL("k:1") <= 0 {
		t.Fatal("TTL 应为正数（jitter 后仍为正）")
	}
}

// TestGetJSON_Miss 未命中返回 found=false empty=false。
func TestGetJSON_Miss(t *testing.T) {
	_, c := newTestCache(t)
	ctx := context.Background()

	var got map[string]any
	found, empty := c.GetJSON(ctx, "no:key", &got)
	if found || empty {
		t.Fatalf("期望未命中 found=false empty=false，实际 found=%v empty=%v", found, empty)
	}
}

// TestSetEmpty_GetEmpty 空值占位命中返回 empty=true。
func TestSetEmpty_GetEmpty(t *testing.T) {
	srv, c := newTestCache(t)
	ctx := context.Background()

	c.SetEmpty(ctx, "e:1")
	var got map[string]any
	found, empty := c.GetJSON(ctx, "e:1", &got)
	if found || !empty {
		t.Fatalf("期望命中空值 found=false empty=true，实际 found=%v empty=%v", found, empty)
	}
	if !srv.Exists("e:1") {
		t.Fatal("空值占位未写入")
	}
}

// TestDelete 删除后 key 不存在。
func TestDelete(t *testing.T) {
	srv, c := newTestCache(t)
	ctx := context.Background()

	c.SetJSON(ctx, "d:1", map[string]int{"a": 1}, time.Minute)
	c.Delete(ctx, "d:1")
	if srv.Exists("d:1") {
		t.Fatal("Delete 后 key 仍存在")
	}
}

// TestDeletePattern 按模式删除只删匹配项，不误删其他 key。
func TestDeletePattern(t *testing.T) {
	srv, c := newTestCache(t)
	ctx := context.Background()

	c.SetJSON(ctx, "cf:comment:1001:1:20", map[string]int{}, time.Minute)
	c.SetJSON(ctx, "cf:comment:1001:2:20", map[string]int{}, time.Minute)
	c.SetJSON(ctx, "cf:comment:1002:1:20", map[string]int{}, time.Minute)

	c.DeletePattern(ctx, "cf:comment:1001:*")

	if srv.Exists("cf:comment:1001:1:20") || srv.Exists("cf:comment:1001:2:20") {
		t.Fatal("模式删除后 1001 的 key 仍存在")
	}
	if !srv.Exists("cf:comment:1002:1:20") {
		t.Fatal("不应误删 1002 的 key")
	}
}

// TestNilClient_Noop nil client 时全部方法静默 no-op，不 panic、视为未命中。
func TestNilClient_Noop(t *testing.T) {
	c := cache.NewCache(nil)
	ctx := context.Background()

	var got map[string]any
	found, empty := c.GetJSON(ctx, "k", &got)
	if found || empty {
		t.Fatalf("nil client 应视为未命中，实际 found=%v empty=%v", found, empty)
	}
	// 以下调用不应 panic
	c.SetJSON(ctx, "k", map[string]int{"a": 1}, time.Minute)
	c.SetEmpty(ctx, "k")
	c.Delete(ctx, "k")
	c.DeletePattern(ctx, "k:*")
}
