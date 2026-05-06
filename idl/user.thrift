namespace go user
include "base.thrift"

struct RegisterReq {
    1: string username
    2: string password
}

struct RegisterResp {
    1: base.Response resp
}

struct LoginReq {
    1: string username
    2: string password
}

struct LoginResp {
    1: base.Response resp
    2: string token
}

struct UploadResumeReq {
    1: string user_id
    2: string file_name
    3: binary file_content
    4: string content_type
}

struct UploadResumeResp {
    1: base.Response resp
    2: string resume_id
    3: string file_url
}

struct CheckResumeStatusReq {
    1: string user_id
    2: string resume_id
}

struct CheckResumeStatusResp {
    1: base.Response resp
    2: i32 parse_status
}

service UserService {
    RegisterResp Register(1: RegisterReq req)
    LoginResp Login(1: LoginReq req)
    UploadResumeResp UploadResume(1: UploadResumeReq req)
    CheckResumeStatusResp CheckResumeStatus(1: CheckResumeStatusReq req)
}