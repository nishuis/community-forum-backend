// test/configs —— configs 包黑盒单测（package configs_test）。
// 覆盖环境变量覆盖 YAML 配置的行为：优先级 环境变量 > YAML，
// 非法 int 环境变量与未设置环境变量时应保持 YAML 原值。
package configs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nishuis/community-forum-backend/configs"
)

// writeMinConfig 在临时目录写入一份覆盖全部必填项的最小配置，返回文件路径。
func writeMinConfig(t *testing.T) string {
	t.Helper()
	content := `server:
  port: 8080
mysql:
  host: localhost
  port: 3306
  dbname: community_forum
  username: root
  password: "123456"
  charset: utf8mb4
jwt:
  secret: "yaml-secret"
  access_exp_hour: 2
  refresh_exp_day: 7
redis:
  enable: true
  host: 127.0.0.1
  port: 6379
  password: ""
log:
  level: info
  output: stdout
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入临时配置文件失败: %v", err)
	}
	return path
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	path := writeMinConfig(t)

	// 设置应覆盖的整型与字符串环境变量
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DB_HOST", "mysql")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("DB_USER", "forum_user")
	t.Setenv("DB_PASSWORD", "forum_password")
	t.Setenv("DB_NAME", "forum_db")
	t.Setenv("REDIS_HOST", "redis")
	t.Setenv("JWT_SECRET", "env-secret")
	t.Setenv("JWT_ACCESS_EXP_HOUR", "24")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := configs.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}

	// 环境变量覆盖生效
	if cfg.Server.Port != 9090 {
		t.Errorf("SERVER_PORT 未生效, got %d, want 9090", cfg.Server.Port)
	}
	if cfg.Mysql.Host != "mysql" || cfg.Mysql.Port != 3307 {
		t.Errorf("DB_HOST/DB_PORT 未生效, got %s:%d, want mysql:3307", cfg.Mysql.Host, cfg.Mysql.Port)
	}
	if cfg.Mysql.Username != "forum_user" || cfg.Mysql.Password != "forum_password" || cfg.Mysql.Dbname != "forum_db" {
		t.Errorf("DB_USER/DB_PASSWORD/DB_NAME 未生效, got %s/%s/%s",
			cfg.Mysql.Username, cfg.Mysql.Password, cfg.Mysql.Dbname)
	}
	if cfg.Redis.Host != "redis" {
		t.Errorf("REDIS_HOST 未生效, got %s, want redis", cfg.Redis.Host)
	}
	if cfg.Jwt.Secret != "env-secret" || cfg.Jwt.AccessExpHour != 24 {
		t.Errorf("JWT_SECRET/JWT_ACCESS_EXP_HOUR 未生效, got %s/%d", cfg.Jwt.Secret, cfg.Jwt.AccessExpHour)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("LOG_LEVEL 未生效, got %s, want debug", cfg.Log.Level)
	}

	// 未被环境变量覆盖的字段保持 YAML 原值
	if cfg.Mysql.Charset != "utf8mb4" {
		t.Errorf("未覆盖字段 charset 被改动, got %q", cfg.Mysql.Charset)
	}
	if cfg.Jwt.RefreshExpDay != 7 || cfg.Redis.Port != 6379 {
		t.Errorf("未覆盖字段被改动, got refresh=%d redis_port=%d", cfg.Jwt.RefreshExpDay, cfg.Redis.Port)
	}
}

func TestLoadConfig_InvalidIntEnvIgnored(t *testing.T) {
	path := writeMinConfig(t)

	// 非数字端口应被忽略，保留 YAML 值
	t.Setenv("DB_PORT", "not-a-number")
	t.Setenv("SERVER_PORT", "abc")

	cfg, err := configs.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}
	if cfg.Mysql.Port != 3306 {
		t.Errorf("非法 DB_PORT 未被忽略, got %d, want 3306", cfg.Mysql.Port)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("非法 SERVER_PORT 未被忽略, got %d, want 8080", cfg.Server.Port)
	}
}

func TestLoadConfig_NoEnv(t *testing.T) {
	path := writeMinConfig(t)

	// 未设置任何环境变量（当前进程相关变量清空），返回值应与 YAML 完全一致
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("SERVER_PORT", "")

	cfg, err := configs.LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}
	if cfg.Mysql.Host != "localhost" || cfg.Mysql.Port != 3306 || cfg.Mysql.Username != "root" {
		t.Errorf("空环境变量影响了 YAML 值, got %s:%d/%s",
			cfg.Mysql.Host, cfg.Mysql.Port, cfg.Mysql.Username)
	}
	if cfg.Jwt.Secret != "yaml-secret" || cfg.Server.Port != 8080 {
		t.Errorf("空环境变量影响了 YAML 值, got secret=%s port=%d", cfg.Jwt.Secret, cfg.Server.Port)
	}
}

// TestLoadConfig_MissingFile 文件不存在时应返回错误（回退逻辑在 main.go，不在 configs 包内）。
func TestLoadConfig_MissingFile(t *testing.T) {
	if _, err := configs.LoadConfig(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Error("加载不存在的配置文件应返回错误")
	}
}
