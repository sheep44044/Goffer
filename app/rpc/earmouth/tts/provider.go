package tts

import (
	"context"
	"fmt"
	"log"
)

// TTSProvider 语音合成抽象接口。
// 接收一段文本，返回合成的音频字节流（PCM 或 OPUS）。
// 调用方在消费完返回的 channel 后，实现方负责关闭它。
type TTSProvider interface {
	SynthesizeStream(ctx context.Context, text string) (<-chan []byte, error)
}

// NewTTSProvider 根据 provider 名称构造对应实现
func NewTTSProvider(providerName string) (TTSProvider, error) {
	switch providerName {
	case "mock":
		return &MockTTSProvider{}, nil
	default:
		return &MockTTSProvider{}, nil
	}
}

// ===================== Mock 实现 =====================

// MockTTSProvider 模拟语音合成。
// 将输入文本包装为一段假音频字节返回，方便前后端联调。
type MockTTSProvider struct{}

func (m *MockTTSProvider) SynthesizeStream(ctx context.Context, text string) (<-chan []byte, error) {
	out := make(chan []byte, 1)

	go func() {
		defer close(out)

		// 模拟合成延迟
		mockAudio := []byte(fmt.Sprintf("[TTS-Mock-Audio] %s", text))

		select {
		case out <- mockAudio:
			log.Printf("[MockTTS] 合成完成: 文本长度=%d, 音频大小=%d", len(text), len(mockAudio))
		case <-ctx.Done():
			log.Printf("[MockTTS] Context 取消，合成中断")
		}
	}()

	return out, nil
}
