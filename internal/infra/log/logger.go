package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/nishuis/community-forum-backend/configs"
)

// InitLogger 初始化全局 slog logger。
// level 支持 debug / info / warn / error（不区分大小写，默认 info）；
// output 支持 stdout（默认）或日志文件路径（追加写）。
func InitLogger(cfg *configs.Config) error {
	// 1.解析日志级别
	level, err := parseLevel(cfg.Log.Level)
	if err != nil {
		return err
	}

	// 2.确定输出目标
	var out io.Writer = os.Stdout
	output := strings.TrimSpace(cfg.Log.Output)
	if output != "" && !strings.EqualFold(output, "stdout") {
		f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("打开日志输出文件失败: %w", err)
		}
		out = f
	}

	// 3.设置全局 logger（JSON 结构化输出）
	handler := slog.NewJSONHandler(out, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
	return nil
}

// parseLevel 将字符串日志级别解析为 slog.Level
func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("不支持的日志级别: %q（可选 debug/info/warn/error）", s)
	}
}
