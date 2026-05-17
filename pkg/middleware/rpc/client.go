package middleware

import (
	"context"

	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/pkg/klog"

	"Goffer/kitex_gen/agent"
)

func ClientMiddleware(next endpoint.Endpoint) endpoint.Endpoint {
	return func(ctx context.Context, req, resp interface{}) (err error) {
		klog.Infof("RPC Client 发起请求: %v", req)

		// 执行真实的 RPC 调用
		err = next(ctx, req, resp)

		// 如果发生错误（包括超时、被限流、或者触发了熔断）
		if err != nil {
			klog.Errorf("RPC Client 发生错误 (准备尝试降级): %v", err)

			// 🔥 降级逻辑 (Fallback)
			// 因为 resp 是 interface{} 类型，我们需要用类型断言判断当前是在调哪个接口
			switch r := resp.(type) {

			// 假设这是调用 RAG 知识库检索的接口
			case *agent.RetrieveResp:
				// 塞入兜底的上下文数据
				r.Contexts = []string{"(系统提示: 当前知识检索服务拥挤，本条回复仅依赖大模型基础知识)"}
				klog.Infof("触发 RetrieveResp 降级成功！")
				return nil // ⚠️ 返回 nil，屏蔽错误！上层网关会认为请求成功，只是拿到了兜底数据

			// 假设这是调用流式对话的接口
			case *agent.ChatStreamResp:
				r.Chunk = "抱歉，面试官（AI）目前正在思考人生，请稍后再试..."
				klog.Infof("触发 ChatStreamResp 降级成功！")
				return nil

			// 对于其他没有配置降级策略的接口，原样抛出错误
			default:
				klog.Warnf("未匹配到降级策略，向上层抛出错误")
				return err
			}
		}

		klog.Infof("RPC Client 响应成功: %v", resp)
		return nil
	}
}
