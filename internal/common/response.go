package common

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// 常量定义
const (
	errJSONMarshalFailed = "结果转换失败: %v"
)

// 预定义常见错误响应
var (
	commonErrorTexts = map[string]string{
		"json_failed":       "JSON序列化失败",
		"invalid_params":    "参数无效",
		"connection_failed": "连接失败",
		"timeout":           "操作超时",
		"not_found":         "资源未找到",
	}
)

// CreateSuccessResponse 创建成功响应结果
func CreateSuccessResponse(data any) (*mcp.CallToolResultFor[any], error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return &mcp.CallToolResultFor[any]{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(errJSONMarshalFailed, err)}},
		}, nil
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonData)}},
	}, nil
}

// CreateErrorResponse 创建错误响应结果
func CreateErrorResponse(format string, args ...any) (*mcp.CallToolResultFor[any], error) {
	var message string

	// 对于无参数的常见错误，使用预定义文本
	if len(args) == 0 {
		if commonText, exists := commonErrorTexts[format]; exists {
			message = commonText
		} else {
			message = format
		}
	} else {
		// 对于需要格式化的错误，仍使用fmt.Sprintf
		message = fmt.Sprintf(format, args...)
	}

	return &mcp.CallToolResultFor[any]{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}, nil
}

// CreateErrorResponseWithKey 使用预定义错误键的快速错误响应
func CreateErrorResponseWithKey(errorKey string) (*mcp.CallToolResultFor[any], error) {
	if text, exists := commonErrorTexts[errorKey]; exists {
		return &mcp.CallToolResultFor[any]{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, nil
	}

	// 回退到默认错误
	return CreateErrorResponse("未知错误: %s", errorKey)
}

// CreateSimpleSuccessResponse 创建简单字符串成功响应（避免JSON序列化）
func CreateSimpleSuccessResponse(message string) (*mcp.CallToolResultFor[any], error) {
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}, nil
}

// CreateJSONResponse 创建JSON响应
func CreateJSONResponse(data any) (*mcp.CallToolResultFor[any], error) {
	return CreateSuccessResponse(data)
}
