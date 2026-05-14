package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// 1. 定义结构体并使用 jsonschema tag
// Eino 会通过反射自动将其转换为大模型能理解的 function calling parameters Schema
type WebSearchInput struct {
	Query  string `json:"query" jsonschema:"description=搜索关键词,required=true"`
	Engine string `json:"engine" jsonschema:"description=搜索引擎: duckduckgo/bing,default=duckduckgo"`
	Count  int    `json:"count" jsonschema:"description=返回结果数量,default=5"`
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// 2. 创建并返回 Eino 标准 Tool
func NewWebSearchTool() (tool.BaseTool, error) {
	// utils.InferTool 的正确签名直接接收 toolName 和 toolDesc 作为字符串：
	// InferTool(toolName, toolDesc string, i InvokeFunc, opts ...Option)

	return utils.InferTool(
		"web_search", // 参数 1: 工具名称 (string)
		"搜索互联网获取信息，支持多引擎搜索", // 参数 2: 工具描述 (string)
		webSearchExecute, // 参数 3: 具体的执行函数
	)
}

// 3. 执行入口：接收强类型的输入，返回供模型消费的 string
func webSearchExecute(ctx context.Context, input *WebSearchInput) (string, error) {
	if input.Query == "" {
		return "", fmt.Errorf("缺少query参数")
	}

	engine := input.Engine
	if engine == "" {
		engine = "duckduckgo"
	}

	count := input.Count
	if count <= 0 {
		count = 5
	}

	var results []SearchResult
	var err error

	// 引擎路由
	switch engine {
	case "duckduckgo":
		results, err = searchDuckDuckGo(ctx, input.Query, count)
	default:
		results, err = searchDuckDuckGo(ctx, input.Query, count)
	}

	if err != nil {
		return "", fmt.Errorf("搜索失败: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("未找到与 \"%s\" 相关的结果", input.Query), nil
	}

	// 将结果格式化为大模型易于阅读的文本形式
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("搜索 \"%s\" 的结果 (共%d条):\n\n", input.Query, len(results)))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n   %s\n\n", i+1, r.Title, r.Snippet, r.URL))
	}

	return sb.String(), nil
}

// 4. 提取出的核心 HTTP 请求和解析逻辑 (保持原样，只做微调)
func searchDuckDuckGo(ctx context.Context, query string, count int) ([]SearchResult, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1",
		url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "Logos-AIM-MCP/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var ddgResp struct {
		Abstract      string `json:"Abstract"`
		AbstractURL   string `json:"AbstractURL"`
		AbstractTitle string `json:"AbstractTitle"`
		RelatedTopics []struct {
			Text string `json:"Text"`
			URL  string `json:"FirstURL"`
		} `json:"RelatedTopics"`
		Results []struct {
			Text string `json:"Text"`
			URL  string `json:"FirstURL"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(body, &ddgResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	var results []SearchResult

	if ddgResp.Abstract != "" {
		results = append(results, SearchResult{
			Title:   ddgResp.AbstractTitle,
			URL:     ddgResp.AbstractURL,
			Snippet: ddgResp.Abstract,
		})
	}

	for _, r := range ddgResp.Results {
		if len(results) >= count {
			break
		}
		results = append(results, SearchResult{
			Title:   truncate(r.Text, 80),
			URL:     r.URL,
			Snippet: r.Text,
		})
	}

	for _, r := range ddgResp.RelatedTopics {
		if len(results) >= count {
			break
		}
		if r.Text == "" || r.URL == "" {
			continue
		}
		results = append(results, SearchResult{
			Title:   truncate(r.Text, 80),
			URL:     r.URL,
			Snippet: r.Text,
		})
	}

	if len(results) == 0 {
		results = append(results, SearchResult{
			Title:   query,
			URL:     fmt.Sprintf("https://duckduckgo.com/?q=%s", url.QueryEscape(query)),
			Snippet: fmt.Sprintf("请点击链接查看 \"%s\" 的搜索结果", query),
		})
	}

	return results, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
