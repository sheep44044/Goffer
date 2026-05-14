package tools

import (
	"Goffer/app/rpc/agent/mcp/sandbox"
	"Goffer/app/rpc/agent/mcp/search"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/tool"
)

// ToolRegistry 存放系统中所有初始化好的 Eino 工具
var ToolRegistry map[string]tool.BaseTool

// InitTools 初始化所有可用的工具 (在服务启动时调用一次即可)
func InitTools() error {
	ToolRegistry = make(map[string]tool.BaseTool)

	// 1. 初始化并注册代码执行沙箱
	codeTool, err := sandbox.NewCodeExecutionTool()
	if err != nil {
		return fmt.Errorf("初始化代码执行工具失败: %w", err)
	}
	// 这里的 key 必须和 yaml 预设中 allowed_tools 填写的名字一致
	ToolRegistry["code_execution"] = codeTool

	// 2. 初始化并注册联网搜索
	searchTool, err := search.NewWebSearchTool()
	if err != nil {
		return fmt.Errorf("初始化联网搜索工具失败: %w", err)
	}
	ToolRegistry["web_search"] = searchTool
	log.Printf("[Tools] 🎉 成功加载 %d 个 MCP 工具", len(ToolRegistry))

	return nil
}

// GetToolsByName 根据预设文件 (yaml) 中的工具名列表，返回真实的工具实例数组
// 这个方法是专门给 bot.go 里的 Eino Agent 挂载工具时用的
func GetToolsByName(toolNames []string) []tool.BaseTool {
	var activeTools []tool.BaseTool
	for _, name := range toolNames {
		if t, ok := ToolRegistry[name]; ok {
			activeTools = append(activeTools, t)
		} else {
			log.Printf("[Tools] ⚠️ 警告: YAML 预设中请求了未知的工具 '%s'", name)
		}
	}
	return activeTools
}
