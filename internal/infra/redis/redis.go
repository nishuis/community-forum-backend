// infra/redis/redis.go —— Redis 客户端初始化。
// 设计原则：Redis 是"可丢弃的加速层"，MySQL 是唯一权威数据源。
// 因此初始化采取 fail-open 降级策略：缓存未启用或连接失败时，仅告警并返回 nil，
// 由上层调用方（service 层缓存封装）判空后跳过缓存、直连数据库，绝不影响业务可用性。
package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nishuis/community-forum-backend/configs"
	goredis "github.com/redis/go-redis/v9"
)

// InitRedis 初始化 go-redis 客户端。
// 返回 nil 表示"未启用或不可用"，调用方应降级直连数据库。
func InitRedis(cfg *configs.Config) *goredis.Client {
	// 1.缓存总开关
	if cfg == nil || !cfg.Redis.Enable {
		slog.Warn("redis 缓存未启用（enable=false），跳过初始化，直连数据库")
		return nil
	}

	// 2.创建客户端
	client := goredis.NewClient(&goredis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	// 3.探活，失败则降级
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		slog.Warn("redis 连接失败，降级为直连数据库", "err", err)
		_ = client.Close()
		return nil
	}

	slog.Info("redis 连接成功",
		"addr", fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		"db", cfg.Redis.DB,
	)
	return client
}

// CloseRedis 关闭 redis 连接，nil 安全。
func CloseRedis(client *goredis.Client) {
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		slog.Warn("redis 连接关闭失败", "err", err)
		return
	}
	slog.Info("redis 连接已关闭")
}
