namespace go knowledge
include "base.thrift"

// 题库结构
struct Question {
    1: string question_id
    2: string question_content  // 题目描述
    3: string standard_answer   // 标准答案/采分点
    4: list<string> tags        // 标签（如 ["Golang", "并发"]）
    5: string difficulty        // 难度（如 "Easy", "Medium", "Hard"）
    6: string create_time
}

// 岗位描述(JD)结构
struct JobDescription {
    1: string jd_id
    2: string title             // 岗位名称 (如 "高级 Go 后端工程师")
    3: string company           // 公司名称
    4: string responsibilities
    5: string requirements          // 岗位要求详情
    6: list<string> tags        // 技能要求标签
    7: string create_time
}

struct IngestQuestionReq {
    1: required string question_content // 必填：牛客上爬下来的问题
    2: required string standard_answer  // 必填：八股文标准答案
    // 下面交给 AI 自动生成，前端界面可以不传
    3: optional list<string> tags
    4: optional string difficulty
}

struct IngestQuestionResp {
    1: base.Response resp
    2: string question_id

}

struct UploadQuestionReq {
    1: required string user_id
    2: required string file_name
    3: required binary file_content
    4: required string content_type
}

struct UploadQuestionResp {
    1: base.Response resp
    2: string task_id
    3: string file_url
}

struct IngestJDReq {
    1: required string company          // 必填：如 "字节跳动"
    2: required string title            // 必填：如 "后端开发实习生"
    3: required string responsibilities // 必填：岗位职责（日常干什么）
    4: required string requirements     // 必填：任职要求（需要什么技术）
    // 下面交给 AI 自动生成，提取核心技能点
    5: optional list<string> tags
}

struct IngestJDResp {
    1: base.Response resp
    2: string jd_id
}

struct UploadJDReq {
    1: required string user_id
    2: required string file_name
    3: required binary file_content
    4: required string content_type
}

struct UploadJDResp {
    1: base.Response resp
    2: string task_id
    3: string file_url
}

service KnowledgeService {
    IngestQuestionResp IngestQuestion(1: IngestQuestionReq req)
    UploadQuestionResp UploadQuestion(1: UploadQuestionReq req)

    IngestJDResp IngestJD(1: IngestJDReq req)
    UploadJDResp UploadJD(1: UploadJDReq req)
    // (可选) 你以后还可以在这里加 ListQuestions 等只查 MySQL 的接口供后台展示用
}