# Goffer

*一个基于 WebRTC + Eino Agent 的全双工、可打断、RAG 增强的拟真 AI 面试平台*

---

## 接口文档

完整 API 文档（含请求/响应示例、鉴权方式、WebRTC 信令协议）→ [docs/API.md](docs/API.md)

---

## 快速开始

### 1. 启动基础设施

```bash
make up
```

这会在 Docker 中启动 Etcd, Redis, MySQL, MongoDB, MinIO, Qdrant, Kafka, Jaeger, Prometheus, Grafana, OTel Collector。

### 2. 启动所有微服务

```bash
make run-all
```

或者按依赖顺序逐个启动：

```bash
go run app/rpc/user/main.go &
go run app/rpc/knowledge/main.go &
go run app/rpc/agent/main.go &
go run app/rpc/interview/main.go &
go run app/rpc/earmouth/main.go &
go run app/rpc/media/main.go &
go run app/api/main.go &          # API 网关最后启动 (:8080)
```

停止所有服务：

```bash
make stop
```

### 3. 可观测面板

| 面板 | 地址 |
|------|------|
| API 网关 | `http://localhost:8080` |
| Jaeger UI | `http://localhost:16686` |
| Grafana | `http://localhost:3000` |
| MinIO 控制台 | `http://localhost:9001` |
| Prometheus | `http://localhost:9090` |

### 4. 面试 API 流程

```bash
# 注册 & 登录
curl -X POST http://localhost:8080/api/user/register \
  -H "Content-Type: application/json" \
  -d '{"username":"candidate","password":"123456"}'

TOKEN=$(curl -s -X POST http://localhost:8080/api/user/login \
  -H "Content-Type: application/json" \
  -d '{"username":"candidate","password":"123456"}' | jq -r '.data')

# 上传简历
curl -X POST http://localhost:8080/api/resume/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@resume.pdf"

# 开始面试
curl -X POST http://localhost:8080/api/interview/start \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"resume_id":"YOUR_RESUME_ID"}'
```

文本流式对话通过 SSE 接入。全双工语音通过 WebSocket 信令建立 WebRTC 连接：`ws://localhost:8890/ws`

---

## 项目总览

| 层级 | 组件 |
|------|------|
| **RPC 框架** | Kitex + Thrift IDL |
| **API 网关** | Hertz + SSE |
| **AI 编排** | Eino (DAG + ReAct Agent + MCP Tools) |
| **WebRTC** | Pion (SFU 架构, Track 中转) |
| **消息队列** | Kafka (audio.in / text.in / text.out / audio.out) |
| **缓存与状态** | Redis (FSM 状态机, Token 黑名单, Pub/Sub) |
| **数据库** | MySQL (结构化业务数据), MongoDB (对话日志) |
| **对象存储** | MinIO (简历 PDF, JD CSV) |
| **向量数据库** | Qdrant (RAG 检索) |
| **可观测性** | OpenTelemetry + Zap + Jaeger + Prometheus + Grafana |
| **服务注册** | Etcd |
| **大模型** | 火山引擎 Ark (Doubao) |
| **STT / TTS** | Provider Interface 抽象（Mock / Deepgram / EdgeTTS 可插拔） |

### 目录结构

```
Goffer/
├── app/
│   ├── api/                     # Hertz API 网关 (:8080)
│   │   ├── config/              # 网关配置 + JWT
│   │   ├── handlers/            # HTTP Handler (user / interview / knowledge)
│   │   │   └── pack/            # 统一响应封装
│   │   ├── router/              # 路由注册 + 中间件
│   │   └── rpc/                 # 下游 RPC 客户端初始化
│   └── rpc/
│       ├── user/                # 用户服务 (:8888) — 注册/登录/简历管理
│       ├── knowledge/           # 知识库服务 (:8889) — JD/题库管理 + AI 打标
│       ├── agent/               # AI 大脑 (:8891) — Eino DAG + RAG + MCP 工具
│       ├── interview/           # 面试服务 (:8892) — 房间管理 + FSM + 上下文聚合
│       ├── earmouth/            # 听觉表达 (:8893) — Kafka→STT→text.in / text.out→TTS→audio.out
│       └── media/               # 媒体网关 (:8890) — Pion WebRTC 信令 + 音频中转
├── pkg/                         # 公共组件
│   ├── errno/                   # 统一错误码定义
│   ├── logger/                  # 基于 zap 的结构化日志（klog/hlog 重定向 + TraceID）
│   ├── middleware/               # JWT 鉴权 / 限流 / RPC 中间件
│   ├── jwt/                     # JWT 签发 + Redis 黑名单
│   ├── telemetry/               # OpenTelemetry 初始化
│   ├── snowflake/               # 分布式 ID 生成
│   ├── pdfparser/               # PDF 文本提取
│   └── contextutil/             # Context 工具（UserID 透传 / IP 获取）
├── docker-compose/              # 基础设施 Docker Compose
├── idl/                         # Thrift IDL 定义
├── kitex_gen/                   # Kitex 生成的 RPC 代码
├── Makefile                     # 一键启动/停止脚本
└── go.mod
```

---

## 架构概览

### 全双工语音数据流

```
Browser              Media               EarMouth               Agent              Qdrant
  │                    │                    │                     │                   │
  │── 麦克风 OPUS ─────►│                    │                     │                   │
  │                    │── audio.in(MQ) ───►│                     │                   │
  │                    │                    ├── STT 语音转文字      │                   │
  │                    │                    │── text.in(MQ) ─────►│                   │
  │                    │                    │                     │─── RAG 向量检索 ──►│
  │                    │                    │                     │◄─ 返回相关上下文 ───│
  │                    │                    │◄── text.out(MQ) ────│ (流式生成文本)      │
  │                    │                    ├── TTS 文字转语音  ────│                   │
  │                    │◄── audio.out(MQ) ──│                     │                   │
  │◄─ 扬声器 OPUS ──────│                    │                     │                   │
  │                    │                    │                     │                   │
```

### 打断时序

```
Browser              Media                Redis             Agent             EarMouth
  │                    │                    │                  │                  │
  │── VAD检测到人声 ────►│                    │                  │                  │
  │                    │──DataChannel msg──►│                  │                  │
  │                    │──Publish cancel───►│                  │                  │
  │                    │                    │──subscribe──────►│                  │
  │                    │                    │                  │──ctx.Cancel()    │
  │                    │                    │                  │  LLM 立即停止     │
  │                    │                    │──subscribe─────────────────────────►│
  │                    │                    │                  │  CancelTracker   │
  │                    │                    │                  │  丢弃积压+中断TTS  │
```

---

## 核心亮点

### 级联打断 (Barge-in)

用户开口说话 → 浏览器 VAD 检测 → WebRTC DataChannel 发送 `{"action":"cancel"}` → Media 服务通过 Redis Pub/Sub 毫秒级广播 → Agent 调用 `context.WithCancel` 终止 LLM 推理 → EarMouth 的 `CancelTracker` 双重拦截 Kafka 积压消息并中断正在进行的 TTS 合成。

### 会话保持

后端完全无状态：会话上下文通过 Kafka 流转。Redis FSM (`greeting → tech_foundation → tech_architecture → evaluator`) 管理面试生命周期。WebRTC 断开后携带 `RoomID` 即可无缝重建连接，MongoDB 持久化对话历史。

### 工业级可观测性

重写基于 Uber-go/zap 的结构化日志组件，重定向 Kitex `klog` 和 Hertz `hlog` 至统一内核。OpenTelemetry TraceID 跨网关、跨 RPC、跨 Kafka、跨 Goroutine 全链路透传，零日志孤岛。

### RAG 个性化面试

简历 PDF/图片 → AI 解析 → Eino 文本分割 → Qdrant 向量数据库。JD 题库 → 自动标签 + 难度分级 → 向量化。面试中实时检索，结合 MCP 工具链（WebSearch、CodeSandbox）实现千人千面的追问。

---
