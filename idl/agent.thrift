namespace go agent
include "base.thrift"

// 基础消息结构，用于历史记录
struct Message {
    1: string role     // "user" 或 "assistant"
    2: string content  // 消息内容
}

// 流式对话请求
struct ChatStreamReq {
    1: string session_id      // 会话唯一标识
    2: string user_id         // 用户唯一标识
    3: string message         // 用户本次输入的文本
    4: string fsm_state       // 状态机当前环节 (如 greeting, tech_foundation)
    5: string resume_id       // 关联的简历 ID，用于 RAG 检索
    6: list<Message> history  // 历史对话上下文
}

// 流式对话响应
struct ChatStreamResp {
    1: string chunk           // 大模型生成的增量文本片段
}

struct RetrieveReq {
    1: required string user_id      // 查谁的资料
    2: required string collection   // 查哪个集合
    3: required string query        // 用户的自然语言提问（如："你用过Redis吗"）
    4: optional i32    top_k        // 期望返回几条最相关的内容（默认比如3）

    5: optional string resume_id  // 查简历时必传
    6: optional string difficulty // 查题库时可选
    7: optional list<string> tags        // 查题库/JD时可选
    8: optional string company    // 查JD时可选
}

struct RetrieveResp {
    // 返回纯文本的片段数组，Interview 服务直接拿去拼 Prompt
    1: list<string> contexts
    2: base.Response resp
}

// Agent 服务定义
service AgentService {
    ChatStreamResp ChatStream (1: ChatStreamReq req) (streaming.mode="server")
    RetrieveResp RetrieveContext(1: RetrieveReq req)
}        