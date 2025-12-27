package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Etcd     EtcdConfig     `mapstructure:"etcd"`
	Redis    RedisConfig    `mapstructure:"redis"`
	MySQL    MySQLConfig    `mapstructure:"mysql"`
	MongoDB  MongoDBConfig  `mapstructure:"mongodb"`
	Gateway  GatewayConfig  `mapstructure:"gateway"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type EtcdConfig struct {
	Endpoints    []string      `mapstructure:"endpoints"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	Prefix       string        `mapstructure:"prefix"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type MySQLConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

type MongoDBConfig struct {
	URI        string `mapstructure:"uri"`
	Database   string `mapstructure:"database"`
	Collection string `mapstructure:"collection"`
}

type GatewayConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

type LogConfig struct {
	Level       string   `mapstructure:"level"`
	Encoding    string   `mapstructure:"encoding"`
	OutputPaths []string `mapstructure:"output_paths"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func (c *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Username, c.Password, c.Host, c.Port, c.Database)
}
