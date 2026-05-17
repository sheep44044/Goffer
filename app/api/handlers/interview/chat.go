package interview

import (
	"Goffer/app/api/rpc"
	"Goffer/kitex_gen/interview"
	"Goffer/pkg/logger"
	"context"
	"io"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/sse"
	"go.uber.org/zap"
)

func ChatStream(ctx context.Context, c *app.RequestContext) {
	var req ChatReq
	if err := c.BindAndValidate(&req); err != nil {
		c.JSON(400, map[string]string{"error": "参数错误"})
		return
	}

	// 1. 设置 HTTP 状态码和 Header，准备建立 SSE 连接
	c.SetStatusCode(200)
	stream := sse.NewStream(c)

	logger.InfoCtx(ctx, "收到流式对话请求", zap.String("session_id", req.SessionID), zap.String("content", req.Content))

	streamCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	// 2. 调用底层的 Kitex RPC 流式接口 (补充了被遗漏的 .ChatStream 方法名)
	rpcStream, err := rpc.ChatStream(streamCtx, &interview.ChatReq{
		SessionId: req.SessionID,
		Message:   req.Content, // 映射到 RPC IDL 中定义的字段
	})
	if err != nil {
		logger.ErrorCtx(ctx, "调用内部 RPC 流式接口失败:", zap.Error(err))
		_ = stream.Publish(&sse.Event{
			Event: "message",
			Data:  []byte("（系统提示：AI 面试官暂时走神了，请稍后重试）"),
		})
		_ = stream.Publish(&sse.Event{Event: "done", Data: []byte("[DONE]")})
		return
	}

	// 3. 循环接收 RPC 发来的字，并原封不动地通过 SSE 推给前端
	for {
		resp, err := rpcStream.Recv()
		if err == io.EOF {
			break // 后端微服务说：“我发完了”
		}
		if err != nil {
			logger.ErrorCtx(ctx, "读取 RPC 流中途异常", zap.Error(err))
			break
		}

		// 推送给前端浏览器
		err = stream.Publish(&sse.Event{
			Event: "message",
			Data:  []byte(resp.Chunk),
		})
		if err != nil {
			logger.ErrorCtx(ctx, "SSE 推送前端失败(用户可能已断开)", zap.Error(err))
			break
		}
	}

	// 4. 告知前端本次回答结束标识
	_ = stream.Publish(&sse.Event{Event: "done", Data: []byte("[DONE]")})
}
