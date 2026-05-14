package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// 1. 定义强类型的输入结构体，使用 jsonschema 标签描述供大模型阅读的元信息
type CodeExecutionInput struct {
	Code     string `json:"code" jsonschema:"description=要执行的代码,required=true"`
	Language string `json:"language" jsonschema:"description=编程语言: python/javascript/go,default=python"`
	Timeout  int    `json:"timeout" jsonschema:"description=超时时间(秒),default=30"`
}

// 2. 创建并返回 Eino 标准 Tool
func NewCodeExecutionTool() (tool.BaseTool, error) {
	// 直接传递名称、描述和执行函数，Eino 会自动推导 Schema 并处理序列化
	return utils.InferTool(
		"code_execution",
		"在沙箱环境中执行代码，支持 Python/JavaScript/Go",
		codeExecutionExecute,
	)
}

// 3. 业务执行入口：接收强类型的 CodeExecutionInput，返回纯 string 结果
func codeExecutionExecute(ctx context.Context, input *CodeExecutionInput) (string, error) {
	if input.Code == "" {
		return "", fmt.Errorf("缺少code参数")
	}

	language := input.Language
	if language == "" {
		language = "python"
	}

	timeout := input.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	result, err := executeCode(ctx, input.Code, language, timeout)
	if err != nil {
		// 这里返回的 error 通常指沙箱环境本身的异常（如创建目录失败），大模型会收到这个错误
		// 如果是代码本身的语法错误，executeCode 通常会正常返回，并在 result(stderr) 里体现
		return "", fmt.Errorf("代码执行环境异常: %w", err)
	}

	return result, nil
}

// 4. 核心沙箱执行逻辑 (基本保持你的原样代码，它是非常健壮的 OS 交互层)
func executeCode(ctx context.Context, code, language string, timeoutSec int) (string, error) {
	tmpDir, err := os.MkdirTemp("", "logos-code-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	var cmd *exec.Cmd
	var filename string

	switch strings.ToLower(language) {
	case "python", "python3":
		filename = "main.py"
		if err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(code), 0644); err != nil {
			return "", fmt.Errorf("写入代码文件失败: %w", err)
		}
		cmd = exec.CommandContext(ctx, "python3", filename)
		cmd.Dir = tmpDir

	case "javascript", "js", "node":
		filename = "main.js"
		if err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(code), 0644); err != nil {
			return "", fmt.Errorf("写入代码文件失败: %w", err)
		}
		cmd = exec.CommandContext(ctx, "node", filename)
		cmd.Dir = tmpDir

	case "go":
		filename = "main.go"
		wrappedCode := fmt.Sprintf(`package main

import (
	"fmt"
	"os"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "panic: %%v", r)
			os.Exit(1)
		}
	}()

	%s
}`, indentCode(code))
		if err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(wrappedCode), 0644); err != nil {
			return "", fmt.Errorf("写入代码文件失败: %w", err)
		}
		cmd = exec.CommandContext(ctx, "go", "run", filename)
		cmd.Dir = tmpDir

	default:
		return "", fmt.Errorf("不支持的语言: %s，支持: python/javascript/go", language)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return "", fmt.Errorf("执行超时 (%ds)", timeoutSec)
	case err := <-done:
		stdoutStr := stdout.String()
		stderrStr := stderr.String()
		exitCode := 0

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = -1
			}
		}

		var sb strings.Builder
		if stdoutStr != "" {
			sb.WriteString("=== 输出 ===\n")
			sb.WriteString(stdoutStr)
		}
		if stderrStr != "" {
			sb.WriteString("=== 错误 ===\n")
			sb.WriteString(stderrStr)
		}
		if stdoutStr == "" && stderrStr == "" {
			sb.WriteString("(无输出)")
		}
		if exitCode != 0 {
			sb.WriteString(fmt.Sprintf("\n退出码: %d", exitCode))
		}

		return sb.String(), nil
	}
}

func indentCode(code string) string {
	lines := strings.Split(code, "\n")
	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString("\t")
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}
