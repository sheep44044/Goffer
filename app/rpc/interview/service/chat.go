package service

import (
	"Goffer/app/rpc/interview/svc"
	"Goffer/kitex_gen/interview"
	"context"
	"fmt"
	"io"
	"log"

	"github.com/cloudwego/eino/schema"
)

type ChatService struct {
	svc *svc.ServiceContext
}

func NewChatService(svc *svc.ServiceContext) *ChatService {
	return &ChatService{
		svc: svc,
	}
}

func (s *ChatService) ChatStream(ctx context.Context, req *interview.ChatReq, stream interview.InterviewService_ChatStreamServer) error {
	// ==========================================
	// 1. 获取“战前情报” (内部直接查数据库，不需要走 RPC)
	// ==========================================
	// 假设你把 GetChatContext 的逻辑挪到了 db/logic 里面
	contextInfo, err := s.svc.Repo.GetChatContextInterview(ctx, req.SessionId, req.Message)
	if err != nil {
		return fmt.Errorf("获取上下文失败: %w", err)
	}

	// ==========================================
	// 2. 组装大模型 Prompt
	// ==========================================
	sysPrompt := fmt.Sprintf(`你是一个专业的 AI 面试官。
当前面试环节：%s

参考用户的简历内容如下：%s

请根据上述信息，结合上下文对用户进行追问。要求专业、简练，像真实的面试官一样对话。`, contextInfo.FsmState, contextInfo.RagChunks)

	// 构建 Eino 的标准消息数组 (复用你原来的逻辑)
	messages := []*schema.Message{
		schema.SystemMessage(sysPrompt),
	}
	for _, h := range contextInfo.History {
		if h.Role == "user" {
			messages = append(messages, schema.UserMessage(h.Content))
		} else {
			messages = append(messages, schema.AssistantMessage(h.Content, nil))
		}
	}
	messages = append(messages, schema.UserMessage(req.Message))

	// ==========================================
	// 3. 发起流式请求 (使用你刚封装好的 AIService)
	// ==========================================
	aiStreamReader, err := s.svc.AI.GetChatStream(ctx, messages)
	if err != nil {
		return fmt.Errorf("调用大模型失败: %w", err)
	}
	defer aiStreamReader.Close()

	fullAnswer := ""

	// ==========================================
	// 4. 循环接收大模型的字，并通过 RPC Stream 推给网关
	// ==========================================
	for {
		msg, err := aiStreamReader.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("流式读取异常: %w", err)
		}

		// 重点修复：go-openai 提取流式 chunk 的正确写法
		if len(msg.Choices) > 0 {
			chunk := msg.Choices[0].Delta.Content
			if chunk != "" {
				fullAnswer += chunk

				// 将大模型吐出的片段，通过 RPC 发送给外层的 API 网关
				err = stream.Send(&interview.ChatResp{
					Chunk: chunk,
				})
				if err != nil {
					log.Printf("网关断开连接，停止发送: %v", err)
					break // 如果网关(或者用户前端)断开了，终止大模型推理，节约 Token
				}
			}
		}
	}
	// ==========================================
	// 5. 内部异步调用数据库，落地“战后记忆”
	// ==========================================
	go func(sid, userMsg, aiMsg string) {
		bgCtx := context.Background()
		// 直接调用数据库层保存，不再走网络 RPC
		err := s.svc.Repo.SaveChatRecordInterview(bgCtx, sid, userMsg, aiMsg, "2")
		if err != nil {
			log.Printf("异步保存聊天记录失败: %v", err)
		}
	}(req.SessionId, req.Message, fullAnswer)

	return nil
}
