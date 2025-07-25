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
	return &mcp.CallToolResultFor[any]{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(format, args...)}},
	}, nil
}

// CreateJSONResponse 创建JSON响应（别名，保持向后兼容）
func CreateJSONResponse(data any) (*mcp.CallToolResultFor[any], error) {
	return CreateSuccessResponse(data)
}
