package config

import (
	"log"

	"github.com/spf13/viper"
)

// 全局配置变量
var AppConfig Config

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

// LoadConfig 加载配置
func LoadConfig(path string) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Unable to decode into struct: %v", err)
	}

	log.Println("Config loaded successfully from", path)
}
