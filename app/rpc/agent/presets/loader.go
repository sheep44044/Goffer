package presets

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// InterviewerPreset 严格对应你的 YAML 配置文件格式
type InterviewerPreset struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Temperature  float32  `yaml:"temperature"`
	SystemPrompt string   `yaml:"system_prompt"`
	AllowedTools []string `yaml:"allowed_tools"`
}

// LoadPreset 从指定路径加载单个面试官的 YAML 配置文件
func LoadPreset(filePath string) (*InterviewerPreset, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取 YAML 文件失败 [%s]: %w", filePath, err)
	}

	var preset InterviewerPreset
	if err := yaml.Unmarshal(data, &preset); err != nil {
		return nil, fmt.Errorf("解析 YAML 失败 [%s]: %w", filePath, err)
	}

	return &preset, nil
}
