package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// CodeExecutionInput 定义强类型的输入结构体，使用 jsonschema 标签描述供大模型阅读的元信息
type CodeExecutionInput struct {
	Code     string `json:"code" jsonschema:"description=要执行的代码,required=true"`
	Language string `json:"language" jsonschema:"description=编程语言: python/javascript/go,default=python"`
	Timeout  int    `json:"timeout" jsonschema:"description=超时时间(秒),default=30"`
}

// NewCodeExecutionTool 创建并返回 Eino 标准 Tool
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
	if timeout > 300 {
		timeout = 300 // 最大 5 分钟，防止 time.Duration 溢出
	}

	result, err := executeCode(ctx, input.Code, language, timeout)
	if err != nil {
		// 这里返回的 error 通常指沙箱环境本身的异常（如创建目录失败），大模型会收到这个错误
		// 如果是代码本身的语法错误，executeCode 通常会正常返回，并在 result(stderr) 里体现
		return "", fmt.Errorf("代码执行环境异常: %w", err)
	}

	return result, nil
}

func executeCode(ctx context.Context, code, language string, timeoutSec int) (string, error) {
	// 1. 创建宿主机临时目录（保持不变）
	tmpDir, err := os.MkdirTemp("", "logos-code-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Docker 挂载必须使用绝对路径
	absTmpDir, err := filepath.Abs(tmpDir)
	if err != nil {
		return "", fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 2. 使用 Context 控制超时（比你原代码中的 select channel 更优雅、更安全）
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	var filename string
	var dockerImage string
	var runCmd []string // 容器内执行的命令和参数

	// 3. 根据语言准备文件、镜像和容器内执行命令
	switch strings.ToLower(language) {
	case "python", "python3":
		filename = "main.py"
		if err := os.WriteFile(filepath.Join(absTmpDir, filename), []byte(code), 0644); err != nil {
			return "", fmt.Errorf("写入代码文件失败: %w", err)
		}
		dockerImage = "python:3.10-slim"
		runCmd = []string{"python", filename}

	case "javascript", "js", "node":
		filename = "main.js"
		if err := os.WriteFile(filepath.Join(absTmpDir, filename), []byte(code), 0644); err != nil {
			return "", fmt.Errorf("写入代码文件失败: %w", err)
		}
		dockerImage = "node:18-slim"
		runCmd = []string{"node", filename}

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
		if err := os.WriteFile(filepath.Join(absTmpDir, filename), []byte(wrappedCode), 0644); err != nil {
			return "", fmt.Errorf("写入代码文件失败: %w", err)
		}
		dockerImage = "golang:1.22-slim"
		runCmd = []string{"go", "run", filename}

	default:
		return "", fmt.Errorf("不支持的语言: %s，支持: python/javascript/go", language)
	}

	// 4. 核心变化：组装简易的 docker run 命令
	// --rm: 容器退出后自动销毁
	// -v: 将宿主机的临时目录挂载到容器的 /workspace
	// -w: 指定工作目录为 /workspace
	// --network none: 断网（防止黑客反弹 Shell、写爬虫、发垃圾邮件）
	// --memory/--memory-swap: 限制内存且禁止 swap（防止内存暴涨压垮服务器）
	// --cpus: 限制 CPU 使用（防止死循环打满宿主机 CPU）
	// --pids-limit: 限制进程数（防止 fork bomb）
	// --security-opt no-new-privileges: 禁止容器内进程提权
	dockerArgs := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/workspace", absTmpDir),
		"-w", "/workspace",
		"--network", "none",
		"--memory", "256m",
		"--memory-swap", "256m",
		"--cpus", "1",
		"--pids-limit", "64",
		"--security-opt", "no-new-privileges",
		dockerImage,
	}
	// 将实际要运行的脚本命令追加到后面
	dockerArgs = append(dockerArgs, runCmd...)

	// 5. 执行命令
	cmd := exec.CommandContext(execCtx, "docker", dockerArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	// 6. 处理执行结果
	stdoutStr := stdout.String()
	stderrStr := stderr.String()
	exitCode := 0

	if err != nil {
		// 如果是 Context 超时引发的错误
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("执行超时 (%ds)", timeoutSec)
		}
		// 如果是代码运行失败（非 0 退出码）
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			// 说明是系统级错误（比如宿主机没装 docker）
			return "", fmt.Errorf("调起 Docker 失败: %w", err)
		}
	}

	// 7. 美化输出（保持你原有的逻辑）
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
