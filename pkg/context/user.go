package context

import (
	"context"
	"errors"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/hertz/pkg/app"
)

// GetUserIDFromGateway 专供【API 网关层】使用
// 从 Hertz 的 RequestContext 中提取 string 类型的 UserID
func GetUserIDFromGateway(c *app.RequestContext) (string, error) {
	uidRaw, exists := c.Get("user_id")
	if !exists {
		return "", errors.New("未鉴权: 请求上下文中无 user_id")
	}

	// 直接断言为 string，因为中间件里存的就是 string
	uidStr, ok := uidRaw.(string)
	if !ok || uidStr == "" {
		return "", errors.New("用户ID格式错误: 期望 string 类型")
	}

	return uidStr, nil
}

// GetUserIDFromRPC 专供【内部微服务层 (Kitex)】使用
// 从标准的 context.Context 通过 metainfo 提取透传的 UserID
func GetUserIDFromRPC(ctx context.Context) (string, error) {
	// metainfo.GetValue 会读取上游服务(或网关)透传过来的 Header/Meta
	uidStr, ok := metainfo.GetValue(ctx, "user_id")
	if !ok || uidStr == "" {
		return "", errors.New("跨服务鉴权失败: RPC 上下文中无 user_id 透传")
	}

	return uidStr, nil
}
