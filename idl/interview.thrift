namespace go interview
include "base.thrift"

struct StartInterviewReq {
    1: string user_id
    2: string resume_id
}

struct StartInterviewResp {
    1: base.Response resp
    2: string session_id
    3: string opening_remark
}

struct ChatReq {
    1: string session_id
    2: string message
}

struct ChatResp {
    1: string chunk  // 每次只推一个字的切片
}

// 定义统一的消息体结构（用于传递历史记录）
struct ChatMessage {
    1: string role    // 角色：通常是 "user" 或 "assistant"
    2: string content // 具体的聊天内容
}

struct GetChatContextReq {
    1: string session_id
    2: string resume_id
    3: string latest_user_msg // ⚠️ 必须传入：底层需要用用户发来的这句话，去 Qdrant 里做 RAG 向量检索
}

struct GetChatContextResp {
    1: base.Response resp
    2: string fsm_state           // 当前的面试环节（从 Redis 中读取，例如 "project_deep_dive"）
    3: list<ChatMessage> history  // 最近 N 轮的历史对话记录（从 MongoDB 中拉取）
    4: string rag_chunks    // 检索出的最高相关度简历切片（从 Qdrant 中拉取）
}

struct ResumeSessionReq {
    1: string session_id
}

struct ResumeSessionResp {
    1: base.Response resp
    2: string fsm_state
    3: i32 round
    4: list<ChatMessage> history
}

service InterviewService {
    // 初始化面试房间与 FSM
    StartInterviewResp StartInterview(1: StartInterviewReq req)
    // 处理一轮对话（检索 RAG + 思考 + 存入 MongoDB）

    ChatResp ChatStream(1: ChatReq req) (streaming.mode="server")

    GetChatContextResp GetChatContext(1: GetChatContextReq req)

    ResumeSessionResp ResumeSession(1: ResumeSessionReq req)
}