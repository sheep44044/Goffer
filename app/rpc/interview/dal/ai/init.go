package ai

import (
	"Goffer/app/rpc/interview/config"
	"context"

	"github.com/cloudwego/eino/schema"
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

// GetChatStream 封装流式对话的具体逻辑，对上层屏蔽 SDK 细节
func (s *AIService) GetChatStream(ctx context.Context, messages []*schema.Message) (*openai.ChatCompletionStream, error) {

	// 将 Eino 的格式转换为 go-openai 的格式
	var oaiMessages []openai.ChatCompletionMessage
	for _, msg := range messages {
		var role string
		switch msg.Role {
		case schema.System:
			role = openai.ChatMessageRoleSystem
		case schema.Assistant:
			role = openai.ChatMessageRoleAssistant
		default:
			role = openai.ChatMessageRoleUser
		}

		oaiMessages = append(oaiMessages, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:    s.cfg.VolcEngine.ChatModelID,
		Messages: oaiMessages,
		Stream:   true, // 开启流式输出
	}

	return s.client.CreateChatCompletionStream(ctx, req)
}
