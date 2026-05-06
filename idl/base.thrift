namespace go base

struct Response {
    1: i64 code
    2: string message
}

enum InterviewStatus {
    IDLE = 0,
    SELF_INTRO = 1,    // 自我介绍
    PROJECT_QUERY = 2, // 项目深挖
    TECH_ASK = 3,      // 技术考察
    ENDING = 4         // 结束总结
}