package stt

import (
	"context"
	"fmt"
	"log"
)

// STTProvider 语音识别抽象接口。
// 接收音频字节流（OPUS/PCM），返回识别出的完整句子。
// 调用方负责关闭 audioStream，实现方在 audioStream 关闭后应关闭返回的 string channel。
type STTProvider interface {
	TranscribeStream(ctx context.Context, audioStream <-chan []byte) (<-chan string, error)
}

// NewSTTProvider 根据 provider 名称构造对应实现
func NewSTTProvider(providerName string) (STTProvider, error) {
	switch providerName {
	case "mock":
		return &MockSTTProvider{}, nil
	default:
		return &MockSTTProvider{}, nil
	}
}

// ===================== Mock 实现 =====================

// MockSTTProvider 模拟语音识别。
// 每收到 50 个音频块输出一条假句子，方便前后端联调。
type MockSTTProvider struct{}

func (m *MockSTTProvider) TranscribeStream(ctx context.Context, audioStream <-chan []byte) (<-chan string, error) {
	out := make(chan string, 10)

	go func() {
		defer close(out)
		chunkCount := 0

		for {
			select {
			case _, ok := <-audioStream:
				if !ok {
					log.Printf("[MockSTT] 音频流已关闭，共处理 %d 块", chunkCount)
					return
				}
				chunkCount++

				// 每 50 个 RTP 音频包（约 1 秒 OPUS 20ms 帧）输出一条模拟语句
				if chunkCount%50 == 0 {
					sentence := fmt.Sprintf("[Mock-STT-句子-%d] 模拟识别结果：第 %d 段语音", chunkCount/50, chunkCount/50)
					select {
					case out <- sentence:
						log.Printf("[MockSTT] 输出模拟句子: %s", sentence)
					case <-ctx.Done():
						return
					}
				}

			case <-ctx.Done():
				log.Printf("[MockSTT] Context 取消，停止识别")
				return
			}
		}
	}()

	return out, nil
}
