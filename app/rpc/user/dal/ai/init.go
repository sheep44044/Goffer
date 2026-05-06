package ai

import (
	"Goffer/app/rpc/user/config"

	"github.com/sashabaranov/go-openai"
)

type AIService struct {
	client *openai.Client
	cfg    *config.Config
}

// NewAIService 构造函数，准备注入到 ServiceContext
func NewAIService(cfg *config.Config) *AIService {
	aiConfig := openai.DefaultConfig(cfg.VolcEngine.Key)
	aiConfig.BaseURL = cfg.VolcEngine.BaseURL

	return &AIService{
		client: openai.NewClientWithConfig(aiConfig),
		cfg:    cfg,
	}
}
