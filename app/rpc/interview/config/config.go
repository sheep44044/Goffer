package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App        AppConfig                  `mapstructure:"app"`
	Server     ServerConfig               `mapstructure:"server"`
	Etcd       EtcdConfig                 `mapstructure:"etcd"`
	Service    ServiceConfig              `mapstructure:"service"`
	Redis      RedisConfig                `mapstructure:"redis"`
	MongoDB    MongoDBConfig              `mapstructure:"mongodb"`
	RpcClients map[string]RpcClientConfig `mapstructure:"rpc_clients"`
	Trace      TraceConfig                `mapstructure:"trace"`
}

type AppConfig struct {
	Env string `mapstructure:"env"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type EtcdConfig struct {
	Address string `mapstructure:"address"`
}

type ServiceConfig struct {
	Name string `mapstructure:"name"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type MongoDBConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
	Timeout  int    `mapstructure:"timeout"`
}

type RpcClientConfig struct {
	Name        string `mapstructure:"name"`
	RpcTimeout  int    `mapstructure:"rpc_timeout"`
	ConnTimeout int    `mapstructure:"conn_timeout"`
}

type TraceConfig struct {
	JaegerEndpoint string `mapstructure:"jaeger_endpoint"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./app/rpc/conf") // 查找路径: RPC 的配置目录

	// 同样开启环境变量覆盖机制
	v.SetEnvPrefix("GOFFER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read rpc config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rpc config: %w", err)
	}

	return &cfg, nil
}
