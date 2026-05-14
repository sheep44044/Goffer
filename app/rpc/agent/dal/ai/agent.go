package ai

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

func (s *AIService) ParseResumeToMarkdown(ctx context.Context, fileData []byte, mimeType string) (string, error) {
	// 视觉模型处理比较慢，超时时间建议设置长一点，比如 60 秒
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// 1. 将文件二进制数据转换为 Base64 编码
	base64Data := base64.StdEncoding.EncodeToString(fileData)

	// 2. 构造 Data URI 格式 (例如: data:image/jpeg;base64,/9j/4AAQSkZJRg...)
	imageURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	// 3. 构造极具针对性的 Prompt
	prompt := `你是一个专业的简历解析助手。
请仔细阅读这张简历图片，并将其完整、精确地转换为标准的 Markdown 格式。
要求：
1. 提取所有文字信息，不能有任何遗漏。
2. 完美还原简历的原有层级结构，使用合适的 Markdown 标题（#，##，###）。
3. 如果遇到项目经验或技能清单，请使用无序或有序列表。
4. 如果遇到表格形式的数据，请使用 Markdown 表格表示。
5. 只输出纯 Markdown 内容，不要包含任何如“好的，这是解析结果”之类的废话。`

	// 4. 发起 Vision 请求
	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: s.cfg.VolcEngine.VisionModelID, // 读取你配置的 Vision Model ID
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleUser,
					// 注意这里：使用 MultiContent 传递图文混合消息
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: prompt,
						},
						{
							Type: openai.ChatMessagePartTypeImageURL,
							ImageURL: &openai.ChatMessageImageURL{
								URL: imageURL,
							},
						},
					},
				},
			},
			MaxTokens:   4096, // 简历内容可能很长，把输出 Token 调大
			Temperature: 0.1,  // 简历解析要求极高准确度，将温度调低，避免 AI 幻觉“自己编造经历”
		},
	)

	if err != nil {
		return "", fmt.Errorf("vision model parse failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("api returned no choices")
	}

	// 获取并清理返回的 Markdown 文本
	mdContent := strings.TrimSpace(resp.Choices[0].Message.Content)

	// 有时候 AI 会被系统设定包裹一层 ```markdown ... ```，我们可以顺手去掉它
	mdContent = strings.TrimPrefix(mdContent, "```markdown\n")
	mdContent = strings.TrimPrefix(mdContent, "```\n")
	mdContent = strings.TrimSuffix(mdContent, "\n```")

	return mdContent, nil
}
