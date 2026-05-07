package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 是 API 网关的全局配置
type Config struct {
	App        AppConfig                  `mapstructure:"app"`
	Server     ServerConfig               `mapstructure:"server"`
	JWT        JWTConfig                  `mapstructure:"jwt"`
	Trace      TraceConfig                `mapstructure:"trace"`
	Etcd       EtcdConfig                 `mapstructure:"etcd"`
	RpcClients map[string]RpcClientConfig `mapstructure:"rpc_clients"`
	Redis      RedisConfig                `mapstructure:"redis"`
	VolcEngine VolcEngineConfig           `mapstructure:"volc_engine"`
}

type AppConfig struct {
	Env string `mapstructure:"env"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type JWTConfig struct {
	Issuer         string        `mapstructure:"issuer"`
	SecretKey      string        `mapstructure:"secret_key"`
	ExpirationTime time.Duration `mapstructure:"expiration_time"`
}

type TraceConfig struct {
	JaegerEndpoint string `mapstructure:"jaeger_endpoint"`
}

type EtcdConfig struct {
	Address string `mapstructure:"address"`
}

type RpcClientConfig struct {
	Name        string `mapstructure:"name"`
	RpcTimeout  int    `mapstructure:"rpc_timeout"`
	ConnTimeout int    `mapstructure:"conn_timeout"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type VolcEngineConfig struct {
	Key           string `mapstructure:"key"`
	BaseURL       string `mapstructure:"base_url"`
	ChatModelID   string `mapstructure:"chat_model_id"`
	EmbedModelID  string `mapstructure:"embed_model_id"`
	VisionModelID string `mapstructure:"vision_model_id"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")         // 配置文件名称为 config (不带后缀)
	v.SetConfigType("yaml")           // 明确指定配置文件类型为 yaml
	v.AddConfigPath(".")              // 查找路径: 当前目录
	v.AddConfigPath("./app/api/conf") // 查找路径: API 的配置目录

	// 【微服务高级技巧】允许环境变量覆盖 YAML 配置
	// 例如: 设置环境变量 GOFFER_JWT_SECRET_KEY 可以覆盖 yaml 里的 jwt.secret_key
	v.SetEnvPrefix("GOFFER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read api config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal api config: %w", err)
	}

	return &cfg, nil
}
