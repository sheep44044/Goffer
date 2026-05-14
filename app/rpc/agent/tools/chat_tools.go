package tools

import (
	"Goffer/app/rpc/agent/svc"
	"Goffer/kitex_gen/interview"
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// ChatHistoryInput 定义大模型调用此工具时需要传入的参数
type ChatHistoryInput struct {
	SessionID string `json:"session_id" jsonschema:"description=当前面试的会话ID(SessionID),required=true"`
}

// NewReadChatHistoryTool 实例化读取聊天记录的工具
func NewReadChatHistoryTool(svc *svc.ServiceContext) (tool.BaseTool, error) {
	return utils.InferTool(
		"ReadChatHistorySkill",
		"获取当前面试会话的完整历史聊天记录，主要用于最终的全局评估和打分",
		func(ctx context.Context, input *ChatHistoryInput) (string, error) {

			// 🌟 核心架构体现：通过 RPC 跨服务调用 Interview 模块拿取数据，不直连 DB
			resp, err := svc.InterviewClient.GetChatContext(ctx, &interview.GetChatContextReq{
				SessionId:     input.SessionID,
				LatestUserMsg: "", // 我们只是为了要历史记录，这里传空字符串即可
			})

			// 错误处理：注意这里的返回值是 (string, error)。
			// 如果返回 error，Eino 可能会中断流程；如果我们返回一段带有错误信息的 string，大模型可以自己决定怎么圆场。
			if err != nil {
				return fmt.Sprintf("调用 Interview 服务获取聊天记录失败: %v", err), nil
			}

			// 判空处理
			if resp == nil || len(resp.History) == 0 {
				return "当前会话没有聊天记录。", nil
			}

			// 🌟 将结构化的 History 组装成易于大模型阅读的“剧本格式”
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("【本次面试(SessionID: %s)完整对话记录】\n\n", input.SessionID))

			for _, msg := range resp.History {
				// 根据你 IDL 中的实际定义调整 Role 的判断逻辑
				if msg.Role == "user" {
					sb.WriteString(fmt.Sprintf("候选人: %s\n", msg.Content))
				} else {
					sb.WriteString(fmt.Sprintf("面试官: %s\n", msg.Content))
				}
				sb.WriteString("--------\n")
			}

			// 返回拼接好的纯文本供打分专家 (Evaluator) 阅读
			return sb.String(), nil
		})
}
