package tools

import (
	"Goffer/app/rpc/agent/mcp/sandbox"
	"Goffer/app/rpc/agent/mcp/search"
	"Goffer/pkg/logger"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"go.uber.org/zap"
)

// ToolRegistry 存放系统中所有初始化好的 Eino 工具
var ToolRegistry map[string]tool.BaseTool

// InitTools 初始化所有可用的工具 (在服务启动时调用一次即可)
func InitTools() error {
	ToolRegistry = make(map[string]tool.BaseTool)

	codeTool, err := sandbox.NewCodeExecutionTool()
	if err != nil {
		return fmt.Errorf("初始化代码执行工具失败: %w", err)
	}
	ToolRegistry["code_execution"] = codeTool

	searchTool, err := search.NewWebSearchTool()
	if err != nil {
		return fmt.Errorf("初始化联网搜索工具失败: %w", err)
	}
	ToolRegistry["web_search"] = searchTool

	logger.Info("MCP 工具加载完成", zap.Int("count", len(ToolRegistry)))
	return nil
}

// GetToolsByName 根据预设文件 (yaml) 中的工具名列表，返回真实的工具实例数组
func GetToolsByName(toolNames []string) []tool.BaseTool {
	var activeTools []tool.BaseTool
	for _, name := range toolNames {
		if t, ok := ToolRegistry[name]; ok {
			activeTools = append(activeTools, t)
		} else {
			logger.Warn("YAML 预设中请求了未知的工具", zap.String("tool_name", name))
		}
	}
	return activeTools
}
