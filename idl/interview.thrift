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

// 定义统一的消息体结构（用于传递历史记录）
struct ChatMessage {
    1: string role    // 角色：通常是 "user" 或 "assistant"
    2: string content // 具体的聊天内容
}
// ==========================================
// 接口 1：获取战前情报 (GetChatContext)
// ==========================================
struct GetChatContextReq {
    1: string session_id
    2: string latest_user_msg // ⚠️ 必须传入：底层需要用用户发来的这句话，去 Qdrant 里做 RAG 向量检索
}

struct GetChatContextResp {
    1: base.Response resp
    2: string fsm_state           // 当前的面试环节（从 Redis 中读取，例如 "project_deep_dive"）
    3: list<ChatMessage> history  // 最近 N 轮的历史对话记录（从 MongoDB 中拉取）
    4: string rag_chunks    // 检索出的最高相关度简历切片（从 Qdrant 中拉取）
}

// ==========================================
// 接口 2：战后记忆落地 (SaveChatRecord)
// ==========================================
struct SaveChatRecordReq {
    1: string session_id
    2: string user_msg   // 用户刚才发送的话
    3: string ai_msg     // 大模型流式输出完毕后的完整文案
    4: string next_state // （可选）如果你打算在 Eino Agent 中让大模型自主决定是否进入下一个环节，可以通过这个字段通知底层更新 Redis
}

struct SaveChatRecordResp {
    1: base.Response resp
}

service InterviewService {
    // 初始化面试房间与 FSM
    StartInterviewResp StartInterview(1: StartInterviewReq req)
    // 处理一轮对话（检索 RAG + 思考 + 存入 MongoDB）

    // 1. 战前情报获取
    GetChatContextResp GetChatContext(1: GetChatContextReq req)

    // 2. 战后记忆落地
    SaveChatRecordResp SaveChatRecord(1: SaveChatRecordReq req)
}