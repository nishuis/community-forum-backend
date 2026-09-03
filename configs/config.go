package configs

import (
	"errors"
	"fmt"
	"os"

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
