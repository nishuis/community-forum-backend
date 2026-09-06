package configs

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// 配置
type Config struct {
	Mysql struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Dbname   string `yaml:"dbname"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Charset  string `yaml:"charset"`
	} `yaml:"mysql"`

	Jwt struct {
		Secret        string `yaml:"secret"`
		AccessExpHour int    `yaml:"access_exp_hour"`
		RefreshExpDay int    `yaml:"refresh_exp_day"`
	} `yaml:"jwt"`

	Server struct {
		Port int `yaml:"port"`
	} `yaml:"server"`

	Log struct {
		Level  string `yaml:"level"`  // debug / info / warn / error，默认 info
		Output string `yaml:"output"` // stdout 或日志文件路径，默认 stdout
	} `yaml:"log"`

	Redis struct {
		Enable       bool   `yaml:"enable"` // 缓存总开关，false 时全部走 DB
		Host         string `yaml:"host"`
		Port         int    `yaml:"port"`
		Password     string `yaml:"password"`
		DB           int    `yaml:"db"`
		PoolSize     int    `yaml:"pool-size"`
		MinIdleConns int    `yaml:"min-idle-conns"`
	} `yaml:"redis"`
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("yaml解析配置失败: %w", err)
	}

	// 环境变量覆盖 YAML 值（docker-compose 等场景），优先级：环境变量 > YAML
	applyEnvOverrides(&cfg)

	// 必填配置校验
	if cfg.Mysql.Host == "" || cfg.Mysql.Dbname == "" || cfg.Mysql.Username == "" {
		return nil, errors.New("mysql配置缺失必填项")
	}
	if cfg.Jwt.Secret == "" {
		return nil, errors.New("jwt secret不能为空")
	}
	if cfg.Redis.Host == "" {
		return nil, errors.New("redis host不能为空")
	}

	return &cfg, nil
}

// applyEnvOverrides 用环境变量覆盖 YAML 配置值，命名与 docker-compose.yml 中
// app 服务的 environment 一一对应。变量未设置、值为空或 int 解析失败时保持
// YAML 原值（本地开发不设环境变量时行为与原来完全一致）。
func applyEnvOverrides(cfg *Config) {
	setStr := func(env string, dst *string) {
		if v, ok := os.LookupEnv(env); ok && v != "" {
			*dst = v
		}
	}
	setInt := func(env string, dst *int) {
		if v, ok := os.LookupEnv(env); ok {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
			}
		}
	}

	setInt("SERVER_PORT", &cfg.Server.Port)

	setStr("DB_HOST", &cfg.Mysql.Host)
	setInt("DB_PORT", &cfg.Mysql.Port)
	setStr("DB_USER", &cfg.Mysql.Username)
	setStr("DB_PASSWORD", &cfg.Mysql.Password)
	setStr("DB_NAME", &cfg.Mysql.Dbname)

	setStr("REDIS_HOST", &cfg.Redis.Host)
	setInt("REDIS_PORT", &cfg.Redis.Port)
	setStr("REDIS_PASSWORD", &cfg.Redis.Password)

	setStr("JWT_SECRET", &cfg.Jwt.Secret)
	setInt("JWT_ACCESS_EXP_HOUR", &cfg.Jwt.AccessExpHour)
	setInt("JWT_REFRESH_EXP_DAY", &cfg.Jwt.RefreshExpDay)

	setStr("LOG_LEVEL", &cfg.Log.Level)
}

// BuildMysqlDSN 拼接完整DSN字符串
func (c *Config) BuildMysqlDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.Mysql.Username,
		c.Mysql.Password,
		c.Mysql.Host,
		c.Mysql.Port,
		c.Mysql.Dbname,
		c.Mysql.Charset,
	)
}
