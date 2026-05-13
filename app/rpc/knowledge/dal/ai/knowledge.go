package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sashabaranov/go-openai"
)

type QAMeta struct {
	Tags       []string `json:"tags"`       // 例如: ["Golang", "并发", "Context"]
	Difficulty string   `json:"difficulty"` // 例如: "简单", "中等", "困难"
}

// GenerateTagsAndDifficulty 根据问题和答案生成标签和难度
func (s *AIService) GenerateTagsAndDifficulty(ctx context.Context, question, answer string) (*QAMeta, error) {
	// 设置 30 秒超时：如果 30 秒没生成完，强制取消，报错返回
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 截断过长内容，避免超过 Token 限制
	// 问题通常较短，答案可能较长，可以根据实际情况调整截断比例
	safeQuestion := truncateContent(question, 500)
	safeAnswer := truncateContent(answer, 1500)

	// 拼接成 User Content
	userContent := fmt.Sprintf("问题：\n%s\n\n答案：\n%s", safeQuestion, safeAnswer)

	// 精确的 System Prompt 是让大模型输出标准 JSON 的关键
	systemPrompt := `你是一个专业的题库助手。请根据用户提供的问题和答案，为其提取合适的标签（1到3个）并评估难度（必须是"简单"、"中等"或"困难"之一）。
请务必直接输出JSON格式的数据，不要包含任何其他说明文字，也不要使用Markdown代码块包裹。
期望的JSON格式如下：
{
  "tags": ["标签1", "标签2"],
  "difficulty": "中等"
}`

	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: s.cfg.VolcEngine.ChatModelID,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userContent,
				},
			},
			Temperature: 0.3,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("tags and difficulty generation failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("api returned no choices")
	}

	rawContent := strings.TrimSpace(resp.Choices[0].Message.Content)

	// 清理大模型可能自作主张加上的 Markdown JSON 标记
	rawContent = strings.TrimPrefix(rawContent, "```json")
	rawContent = strings.TrimPrefix(rawContent, "```")
	rawContent = strings.TrimSuffix(rawContent, "```")
	rawContent = strings.TrimSpace(rawContent)
	// 解析 JSON
	var meta QAMeta
	if err := json.Unmarshal([]byte(rawContent), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON. Raw content: %s, Error: %w", rawContent, err)
	}

	return &meta, nil
}

// GenerateJDTags 根据职位描述内容提取核心技能/业务标签
func (s *AIService) GenerateJDTags(ctx context.Context, jdContent string) ([]string, error) {
	// 设置 30 秒超时：如果 30 秒没生成完，强制取消，报错返回
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 截断过长内容，避免超过 Token 限制
	// JD 通常包含较多文字，保留前 2000 个字符通常已足够提取核心要求
	safeContent := truncateContent(jdContent, 2000)

	// 精确的 System Prompt 设定为 HR 角色，引导输出纯净的 JSON
	systemPrompt := `你是一个专业的HR招聘助手。请根据用户提供的职位描述（JD），提取3到5个最核心的技能、工具或业务领域标签（例如："Golang", "Kubernetes", "微服务", "高并发"等）。
请务必直接输出JSON格式的数据，不要包含任何其他说明文字，也不要使用Markdown代码块包裹。
期望的JSON格式如下：
{
  "tags": ["标签1", "标签2", "标签3"]
}`

	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: s.cfg.VolcEngine.ChatModelID,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: safeContent,
				},
			},
			Temperature: 0.3,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("JD tags generation failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("api returned no choices")
	}

	rawContent := strings.TrimSpace(resp.Choices[0].Message.Content)

	// 清理大模型可能自作主张加上的 Markdown JSON 标记
	rawContent = strings.TrimPrefix(rawContent, "```json")
	rawContent = strings.TrimPrefix(rawContent, "```")
	rawContent = strings.TrimSuffix(rawContent, "```")
	rawContent = strings.TrimSpace(rawContent)

	// 声明一个临时的匿名结构体用于解析大模型返回的 JSON
	var meta struct {
		Tags []string `json:"tags"`
	}

	if err := json.Unmarshal([]byte(rawContent), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON. Raw content: %s, Error: %w", rawContent, err)
	}

	// 容错处理：如果 AI 没有返回任何标签，给一个默认的空数组而不是 nil
	if meta.Tags == nil {
		return []string{}, nil
	}

	return meta.Tags, nil
}

func truncateContent(content string, limit int) string {
	if utf8.RuneCountInString(content) <= limit {
		return content
	}
	runes := []rune(content)
	return string(runes[:limit])
}
