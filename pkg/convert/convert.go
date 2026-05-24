package convert

import (
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
	"google.golang.org/protobuf/types/known/structpb"
)

// 辅助函数：安全地从 Eino+Qdrant 的 Metadata 中提取字符串值
func GetStringFromMeta(doc *schema.Document, key string) string {
	if doc.MetaData == nil {
		return ""
	}

	// 先尝试从平铺的 metadata 中直接取值并断言
	if v, ok := doc.MetaData[key].(*structpb.Value); ok {
		return ParseProtobufValue(v)
	}

	// 再尝试从嵌套的 default_metadata_key 中取值
	rawFields, ok := doc.MetaData["default_metadata_key"]
	if !ok {
		return ""
	}

	fields, ok := rawFields.(map[string]*structpb.Value)
	if !ok {
		return ""
	}

	if val, ok := fields[key]; ok {
		return ParseProtobufValue(val)
	}
	return ""
}

// 核心解析器：专门处理 gRPC structpb.Value 到 Go 基础类型的转换
func ParseProtobufValue(val *structpb.Value) string {
	if val == nil {
		return ""
	}

	switch val.GetKind().(type) {
	case *structpb.Value_StringValue:
		return val.GetStringValue()
	case *structpb.Value_NumberValue:
		return fmt.Sprintf("%.0f", val.GetNumberValue()) // 比如 ID 或者是数字难度
	case *structpb.Value_ListValue:
		// 如果是标签数组 ["Golang", "AI"]，拼装成字符串 "Golang, AI"
		list := val.GetListValue().GetValues()
		var res []string
		for _, item := range list {
			if s, ok := item.GetKind().(*structpb.Value_StringValue); ok {
				res = append(res, s.StringValue)
			}
		}
		return strings.Join(res, ", ")
	default:
		return ""
	}
}
