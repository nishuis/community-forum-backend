package log

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/nishuis/community-forum-backend/internal/middleware"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SlowThreshold 慢查询阈值：超过该耗时的 SQL 记为慢查询
const SlowThreshold = 100 * time.Millisecond

// GormLogger 基于 slog 的 GORM 日志适配器，实现 logger.Interface。
// 仅在「SQL执行出错」或「慢查询」时输出，避免压测时每条 SQL 都打日志。
type GormLogger struct {
	level         logger.LogLevel
	slowThreshold time.Duration
}

// NewGormLogger 创建 GORM slog logger
func NewGormLogger(level logger.LogLevel, slowThreshold time.Duration) *GormLogger {
	return &GormLogger{
		level:         level,
		slowThreshold: slowThreshold,
	}
}

// LogMode 切换日志级别（满足 logger.Interface 接口）
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &GormLogger{
		level:         level,
		slowThreshold: l.slowThreshold,
	}
}

// Info 记录 GORM 信息级日志
func (l *GormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Info {
		slog.InfoContext(ctx, msg, "request_id", middleware.GetRequestID(ctx), "args", args)
	}
}

// Warn 记录 GORM 警告级日志
func (l *GormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Warn {
		slog.WarnContext(ctx, msg, "request_id", middleware.GetRequestID(ctx), "args", args)
	}
}

// Error 记录 GORM 错误级日志
func (l *GormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Error {
		slog.ErrorContext(ctx, msg, "request_id", middleware.GetRequestID(ctx), "args", args)
	}
}

// Trace 记录每条 SQL 的执行情况：仅出错或慢查询时输出
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	// SQL执行出错：忽略 RecordNotFound（正常业务查询不到），其余记录错误
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && l.level >= logger.Error {
		slog.ErrorContext(ctx, "SQL 执行出错",
			"request_id", middleware.GetRequestID(ctx),
			"sql", sql,
			"rows", rows,
			"elapsed_ms", elapsed.Milliseconds(),
			"err", err,
		)
		return
	}

	// 慢查询
	if elapsed > l.slowThreshold && l.level >= logger.Warn {
		slog.WarnContext(ctx, "SQL 慢查询",
			"request_id", middleware.GetRequestID(ctx),
			"sql", sql,
			"rows", rows,
			"elapsed_ms", elapsed.Milliseconds(),
		)
	}
}
