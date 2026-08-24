package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config 总配置结构体
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Mysql  MysqlConfig  `mapstructure:"mysql"`
}

// ServerConfig 服务器相关配置
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// MysqlConfig mysql数据库配置
type MysqlConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Dbname   string `mapstructure:"dbname"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Charset  string `mapstructure:"charset"`
}

// Dsn 拼接生成mysql连接字符，内部自动追加驱动必须参数 parseTime、loc
func (m *MysqlConfig) Dsn() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		m.Username,
		m.Password,
		m.Host,
		m.Port,
		m.Dbname,
		m.Charset,
	)
}

// LoadConfig 加载配置文件，返回配置实例
func LoadConfig() *Config {
	viper.SetConfigName("config")    //配置文件名
	viper.SetConfigType("yaml")      //配置文件格式
	viper.AddConfigPath("./configs") //文件位置

	//读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("读取配置文件失败：%v", err))
	}

	var cfg Config
	// 将yaml配置反序列化映射到结构体
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(fmt.Sprintf("解析配置到结构体失败： %v", err))
	}

	return &cfg
}
