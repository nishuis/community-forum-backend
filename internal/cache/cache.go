// cache/cache.go —— 通用 Redis 旁路缓存封装。
// 设计原则：
//  1. 缓存是"可丢弃的加速层"，所有方法对 Redis 故障 fail-open：出错仅记日志，绝不向上抛错影响业务；
//  2. client 为 nil（未启用/连接失败）时所有方法静默 no-op，调用方自然降级直连数据库；
//  3. 空值占位缓存（防穿透）：SetEmpty 存空串，GetJSON 用 empty 返回值标识"命中空值"。
package cache

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// EmptyValueTTL 空值占位缓存的 TTL（防穿透，短 TTL 快速自愈）。
const EmptyValueTTL = 60 * time.Second

// Cache 通用缓存封装；client 为 nil 时全部方法 no-op。
type Cache struct {
	client *goredis.Client
}

// NewCache 构造缓存实例；client 传 nil 表示禁用缓存。
func NewCache(client *goredis.Client) *Cache {
	return &Cache{client: client}
}

// GetJSON 读取并反序列化缓存值到 target。
// 返回 found=true：命中有效数据（target 已填充）；
// 返回 empty=true：命中空值占位（此前缓存了"不存在"，用于防穿透）；
// Redis 出错或数据损坏时：内部记日志并视为未命中（found=false, empty=false），由调用方直连 DB。
func (c *Cache) GetJSON(ctx context.Context, key string, target any) (found bool, empty bool) {
	if c == nil || c.client == nil {
		return false, false
	}

	val, err := c.client.Get(ctx, key).Bytes()
	if err == goredis.Nil {
		return false, false // 未命中
	}
	if err != nil {
		slog.Warn("redis get 失败，按未命中处理", "key", key, "err", err)
		return false, false
	}
	if len(val) == 0 {
		return false, true // 空值占位命中
	}
	if err := json.Unmarshal(val, target); err != nil {
		slog.Warn("缓存数据反序列化失败，删除该 key", "key", key, "err", err)
		_ = c.client.Del(ctx, key).Err()
		return false, false
	}
	return true, false
}

// SetJSON 序列化并写入缓存，TTL 自动加 ±30% 随机抖动（防雪崩）。
// 写入失败仅记日志，不影响业务。
func (c *Cache) SetJSON(ctx context.Context, key string, val any, ttl time.Duration) {
	if c == nil || c.client == nil {
		return
	}
	data, err := json.Marshal(val)
	if err != nil {
		slog.Warn("缓存序列化失败，跳过写入", "key", key, "err", err)
		return
	}
	if err := c.client.Set(ctx, key, data, jitter(ttl)).Err(); err != nil {
		slog.Warn("redis set 失败", "key", key, "err", err)
	}
}

// SetEmpty 写入空值占位缓存（防穿透），存空串，TTL 固定 EmptyValueTTL。
func (c *Cache) SetEmpty(ctx context.Context, key string) {
	if c == nil || c.client == nil {
		return
	}
	if err := c.client.Set(ctx, key, "", EmptyValueTTL).Err(); err != nil {
		slog.Warn("redis 空值占位写入失败", "key", key, "err", err)
	}
}

// Delete 删除单个缓存 key；失败仅记日志。
func (c *Cache) Delete(ctx context.Context, key string) {
	if c == nil || c.client == nil {
		return
	}
	if err := c.client.Del(ctx, key).Err(); err != nil {
		slog.Warn("redis del 失败", "key", key, "err", err)
	}
}

// DeletePattern 按模式批量删除 key（SCAN 游标循环，避免 KEYS 阻塞）；
// 供评论列表等"写操作不知道存在哪些分页键"的场景使用。
func (c *Cache) DeletePattern(ctx context.Context, pattern string) {
	if c == nil || c.client == nil {
		return
	}
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			slog.Warn("redis 批量删除单项失败", "key", iter.Val(), "err", err)
		}
	}
	if err := iter.Err(); err != nil {
		slog.Warn("redis scan 失败", "pattern", pattern, "err", err)
	}
}

// jitter 给 TTL 加 ±30% 随机抖动，防止大量 key 同时过期造成缓存雪崩。
func jitter(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return ttl
	}
	delta := ttl / 3 // ±30%
	return ttl + time.Duration(rand.Int63n(2*int64(delta)+1)) - delta
}
