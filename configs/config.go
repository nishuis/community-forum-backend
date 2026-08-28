package configs

import (
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
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	return &cfg, err
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
