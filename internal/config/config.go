package config

import (
	"log"

	"github.com/google/wire"
	"github.com/spf13/viper"
)

// ProviderSet 导出给 Wire 使用
var ProviderSet = wire.NewSet(NewConfig)

type Config struct {
	System SystemConfig `mapstructure:"system"`
	Server ServerConfig `mapstructure:"server"`
	MySQL  MySQLConfig  `mapstructure:"mysql"`
	Redis  RedisConfig  `mapstructure:"redis"`
	Kafka  KafkaConfig  `mapstructure:"kafka"`
	Etcd   EtcdConfig   `mapstructure:"etcd"`
}

type SystemConfig struct {
	AppName string `mapstructure:"app_name"`
	Env     string `mapstructure:"env"`
	Version string `mapstructure:"version"`
}

type ServerConfig struct {
	HttpPort int `mapstructure:"http_port"`
	GrpcPort int `mapstructure:"grpc_port"`
}

type MySQLConfig struct {
	DSN     string `mapstructure:"dsn"`
	MaxIdle int    `mapstructure:"max_idle"`
	MaxOpen int    `mapstructure:"max_open"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type KafkaConfig struct {
	Brokers []string `mapstructure:"brokers"`
	Topic   string   `mapstructure:"topic"`
}

type EtcdConfig struct {
	Endpoints   []string `mapstructure:"endpoints"`
	DialTimeout int      `mapstructure:"dial_timeout"`
}

// NewConfig 加载配置并返回对象
// 注意：这里的路径 ./configs/config.yaml 是相对于执行命令的目录
// 如果你在 IDE 中运行，请确保工作目录正确
func NewConfig() *Config {
	viper.SetConfigFile("./configs/config.yaml")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	var conf Config
	if err := viper.Unmarshal(&conf); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}

	log.Println("Config loaded successfully")
	return &conf
}
