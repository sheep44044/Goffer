# Goffer API 文档

Base URL: `http://localhost:8080`

## 通用约定

### 响应格式

所有接口统一返回 JSON：

```json
{
  "code": 0,
  "message": "Success",
  "data": {}
}
```

| code | 说明 |
|------|------|
| 0 | 成功 |
| 10001 | 服务内部错误 |
| 10002 | 参数错误 |
| 10003 | 用户已存在 |
| 10004 | 鉴权失败 |
| 10005 | 用户不存在 |
| 10006 | 密码错误 |
| 20001 | 文件上传失败 |
| 20002 | 文件格式不支持 |
| 20004 | 简历不存在 |

### 鉴权

除 `/api/user/register` 和 `/api/user/login` 外，所有接口需要在 Header 中携带 JWT Token：

```
Authorization: Bearer <token>
```

---

## 用户模块

### POST /api/user/register

注册新用户。

**Request Body**

```json
{
  "username": "candidate",
  "password": "123456"
}
```

**Response**

```json
{
  "code": 0,
  "message": "Success",
  "data": null
}
```

### POST /api/user/login

用户登录，返回 JWT Token。

**Request Body**

```json
{
  "username": "candidate",
  "password": "123456"
}
```

**Response**

```json
{
  "code": 0,
  "message": "Success",
  "data": "eyJhbGciOiJIUzI1NiIs..."
}
```

---

## 简历模块 `🔒`

### POST /api/resume/upload

上传简历文件。支持 PDF / JPEG / PNG，最大 10MB。

**Headers**

```
Authorization: Bearer <token>
```

**Request** `multipart/form-data`

| 字段 | 类型 | 说明 |
|------|------|------|
| file | File | 简历文件 |

**Response**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "resumeID": "1234567890",
    "fileURL": "http://localhost:9000/goffer-resumes/xxx.pdf"
  }
}
```

---

## 面试模块 `🔒`

### POST /api/interview/start

开始一场面试，返回 SessionID 和开场白。

**Headers**

```
Authorization: Bearer <token>
```

**Request Body**

```json
{
  "resume_id": "1234567890"
}
```

**Response**

```json
{
  "code": 0,
  "message": "Success",
  "data": "你好，我是本次的面试官。我已经仔细阅读了你的简历，请先做一个简单的两分钟自我介绍吧。"
}
```

### POST /api/interview/chat

流式对话接口（SSE）。

**Headers**

```
Authorization: Bearer <token>
```

**Request Body**

```json
{
  "session_id": "abc123",
  "content": "我曾在字节跳动负责微服务架构设计"
}
```

**Response** `text/event-stream`

```
event: message
data: 好的

event: message
data: ，请详细

event: message
data: 介绍一下你在字节跳动的微服务架构设计经验。

event: done
data: [DONE]
```

前端使用 `EventSource` 或 `fetch` + `ReadableStream` 消费。

---

## 知识库模块 `🔒`

### POST /api/knowledge/jd/upload

批量上传 JD（职位描述）CSV 文件。

**Headers**

```
Authorization: Bearer <token>
```

**Request** `multipart/form-data`

| 字段 | 类型 | 说明 |
|------|------|------|
| file | File | CSV 文件（表头: 公司,岗位,职责,要求,标签） |

**Response**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "taskID": "csv_xxx",
    "fileURL": "http://localhost:9000/goffer-resources/xxx.csv"
  }
}
```

### POST /api/knowledge/jd/ingest

手动录入单条 JD。

**Headers**

```
Authorization: Bearer <token>
```

**Request Body**

```json
{
  "company": "字节跳动",
  "title": "高级 Golang 工程师",
  "responsibilities": "负责微服务架构设计与实现...",
  "requirements": "5年以上Go开发经验，熟悉K8s...",
  "tags": ["Golang", "Kubernetes", "微服务"]
}
```

**Response**

```json
{
  "code": 0,
  "message": "Success",
  "data": { "jd_id": "jd_xxx" }
}
```

### POST /api/knowledge/question/upload

批量上传题库 CSV 文件。

**Headers**

```
Authorization: Bearer <token>
```

**Request** `multipart/form-data`

| 字段 | 类型 | 说明 |
|------|------|------|
| file | File | CSV 文件（表头: 题目,标准答案,难度,标签） |

**Response**

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "taskID": "csv_xxx",
    "fileURL": "http://localhost:9000/goffer-resources/xxx.csv"
  }
}
```

### POST /api/knowledge/question/ingest

手动录入单道面试题。

**Headers**

```
Authorization: Bearer <token>
```

**Request Body**

```json
{
  "question_content": "请解释 Go 中 Channel 的底层实现原理",
  "standard_answer": "Channel 在运行时由 hchan 结构体表示...",
  "tags": ["Golang", "并发"],
  "difficulty": "中等"
}
```

**Response**

```json
{
  "code": 0,
  "message": "Success",
  "data": { "question_id": "q_xxx" }
}
```

---

## WebRTC 信令

### WS /ws

全双工语音面试通过 WebSocket 建立 WebRTC 连接。

**连接地址**

```
ws://localhost:8890/ws
```

**信令消息**

| type | 方向 | 说明 |
|------|------|------|
| `offer` | 前端→后端 | SDP Offer |
| `answer` | 后端→前端 | SDP Answer |
| `ice-candidate` | 双向 | ICE Candidate 交换 |
| `error` | 后端→前端 | 错误通知 |

**消息格式**

```json
{
  "type": "offer",
  "room_id": "xxx",
  "user_id": "xxx",
  "sdp": "v=0\r\no=...",
  "candidate": "candidate:..."
}
```
