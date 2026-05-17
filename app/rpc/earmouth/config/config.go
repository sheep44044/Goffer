package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App     AppConfig     `mapstructure:"app"`
	Server  ServerConfig  `mapstructure:"server"`
	Etcd    EtcdConfig    `mapstructure:"etcd"`
	Service ServiceConfig `mapstructure:"service"`
	Kafka   KafkaConfig   `mapstructure:"kafka"`
	Redis   RedisConfig   `mapstructure:"redis"`
	STT     STTConfig     `mapstructure:"stt"`
	TTS     TTSConfig     `mapstructure:"tts"`
	Trace   TraceConfig   `mapstructure:"trace"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
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

// KafkaConfig 包含 EarMouth 所需的全部 4 个 Topic 及消费者组配置
type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers"`
	AudioInTopic  string   `mapstructure:"audio_in_topic"`
	TextInTopic   string   `mapstructure:"text_in_topic"`
	TextOutTopic  string   `mapstructure:"text_out_topic"`
	AudioOutTopic string   `mapstructure:"audio_out_topic"`
	ConsumerGroup string   `mapstructure:"consumer_group"`
}

// STTConfig 语音识别服务商配置（预留扩展）
type STTConfig struct {
	Provider string `mapstructure:"provider"` // "mock", "deepgram", "whisper"
	APIKey   string `mapstructure:"api_key"`
	Language string `mapstructure:"language"` // "zh-CN", "en-US"
}

// TTSConfig 语音合成服务商配置（预留扩展）
type TTSConfig struct {
	Provider string `mapstructure:"provider"` // "mock", "edgetts", "volcengine"
	APIKey   string `mapstructure:"api_key"`
	Voice    string `mapstructure:"voice"`    // 音色参数
	Language string `mapstructure:"language"` // "zh-CN", "en-US"
}

type TraceConfig struct {
	JaegerEndpoint string `mapstructure:"jaeger_endpoint"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./app/rpc/conf")

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
