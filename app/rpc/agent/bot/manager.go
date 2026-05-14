package bot

import (
	"Goffer/app/rpc/agent/presets"
	"Goffer/app/rpc/agent/svc"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/cloudwego/eino/schema"
)

type BotManager struct {
	svc  *svc.ServiceContext
	bots map[string]*InterviewBot // 存储编译好的 Eino 图执行器
	mu   sync.RWMutex             // 读写锁，保障高并发下的并发安全
}

var (
	managerInstance *BotManager
	managerOnce     sync.Once
)

// InitBotManager 初始化 Manager 单例
func InitBotManager(svc *svc.ServiceContext) *BotManager {
	managerOnce.Do(func() {
		managerInstance = &BotManager{
			svc:  svc,
			bots: make(map[string]*InterviewBot),
		}
	})
	return managerInstance
}

// GetBotManager 获取 Manager 单例
func GetBotManager() *BotManager {
	return managerInstance
}

// LoadAllPresets 批量加载预设文件并注册 Bot（相当于 Logos 的 RegisterPresetAgent 批量版）
func (m *BotManager) LoadAllPresets() {
	// 定义你要加载的面试官 YAML 路径
	presetFiles := []string{
		"presets/hr.yaml",
		"presets/tech_foundation.yaml",
		"presets/tech_architecture.yaml",
		"presets/evaluator.yaml",
	}

	for _, file := range presetFiles {
		preset, err := presets.LoadPreset(file)
		if err != nil {
			log.Printf("[BotManager] ⚠️ 加载预设文件 %s 失败: %v", file, err)
			continue
		}

		// 根据预设组装 Eino 智能体
		bot, err := NewInterviewBot(preset, m.svc)
		if err != nil {
			log.Printf("[BotManager] ⚠️ 初始化面试官 Bot [%s] 失败: %v", preset.Name, err)
			continue
		}

		// 注册到 Manager 中
		err = m.RegisterBot(preset.Name, bot)
		if err != nil {
			log.Printf("[BotManager] ⚠️ 注册面试官 Bot [%s] 失败: %v", preset.Name, err)
		}
	}

	log.Printf("[BotManager] 🎉 所有预设加载完毕，当前可用面试官数量: %d", len(m.ListBots()))
}

// RegisterBot 注册单个 Bot，加写锁
func (m *BotManager) RegisterBot(name string, bot *InterviewBot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.bots[name]; exists {
		return fmt.Errorf("bot with name %s already exists", name)
	}

	m.bots[name] = bot
	log.Printf("[BotManager] 成功注册面试官: %s", name)
	return nil
}

// GetBot 获取单个 Bot，加读锁
func (m *BotManager) GetBot(name string) (*InterviewBot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bot, exists := m.bots[name]
	if !exists {
		return nil, fmt.Errorf("面试官 %s 不存在，请检查 YAML 预设是否加载", name)
	}

	return bot, nil
}

// ListBots 获取所有已注册的面试官名字列表
func (m *BotManager) ListBots() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.bots))
	for name := range m.bots {
		names = append(names, name)
	}
	return names
}

// 直接通过 Manager 发起流式对话 (类似 Logos 的 ChatStreamWithAgent)
// 这样在 Chat 业务层，你甚至都不需要先 GetBot，直接一步到位调用即可。
func (m *BotManager) StreamAnswer(ctx context.Context, botName string, input BotInput) (*schema.StreamReader[*schema.Message], error) {
	bot, err := m.GetBot(botName)
	if err != nil {
		return nil, err
	}
	// 调用具体 Bot 的内部执行流
	return bot.StreamAnswer(ctx, input)
}
