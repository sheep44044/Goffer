package bot

import (
	"Goffer/app/rpc/agent/presets"
	"Goffer/app/rpc/agent/svc"
	"Goffer/pkg/logger"
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

type BotManager struct {
	svc  *svc.ServiceContext
	bots map[string]*InterviewBot
	mu   sync.RWMutex
}

var (
	managerInstance *BotManager
	managerOnce     sync.Once
)

func InitBotManager(svc *svc.ServiceContext) *BotManager {
	managerOnce.Do(func() {
		managerInstance = &BotManager{
			svc:  svc,
			bots: make(map[string]*InterviewBot),
		}
	})
	return managerInstance
}

func GetBotManager() *BotManager {
	return managerInstance
}

func (m *BotManager) LoadAllPresets() {
	presetFiles := []string{
		"presets/hr.yaml",
		"presets/tech_foundation.yaml",
		"presets/tech_architecture.yaml",
		"presets/evaluator.yaml",
	}

	for _, file := range presetFiles {
		preset, err := presets.LoadPreset(file)
		if err != nil {
			logger.Warn("加载预设文件失败", zap.String("file", file), zap.Error(err))
			continue
		}

		bot, err := NewInterviewBot(preset, m.svc)
		if err != nil {
			logger.Warn("初始化面试官 Bot 失败", zap.String("bot_name", preset.Name), zap.Error(err))
			continue
		}

		err = m.RegisterBot(preset.Name, bot)
		if err != nil {
			logger.Warn("注册面试官 Bot 失败", zap.String("bot_name", preset.Name), zap.Error(err))
		}
	}

	logger.Info("所有预设加载完毕", zap.Int("bot_count", len(m.ListBots())))
}

func (m *BotManager) RegisterBot(name string, bot *InterviewBot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.bots[name]; exists {
		return fmt.Errorf("bot with name %s already exists", name)
	}

	m.bots[name] = bot
	logger.Info("成功注册面试官", zap.String("bot_name", name))
	return nil
}

func (m *BotManager) GetBot(name string) (*InterviewBot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bot, exists := m.bots[name]
	if !exists {
		return nil, fmt.Errorf("面试官 %s 不存在，请检查 YAML 预设是否加载", name)
	}

	return bot, nil
}

func (m *BotManager) ListBots() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.bots))
	for name := range m.bots {
		names = append(names, name)
	}
	return names
}

func (m *BotManager) StreamAnswer(ctx context.Context, botName string, input BotInput) (*schema.StreamReader[*schema.Message], error) {
	bot, err := m.GetBot(botName)
	if err != nil {
		return nil, err
	}
	return bot.StreamAnswer(ctx, input)
}
