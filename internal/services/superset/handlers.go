package superset

import (
	"context"
	"strconv"

	"mcp-server/internal/common"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// 工具参数结构体
type ListDatabasesParams struct{}

type ExecuteSQLParams struct {
	SQL        string `json:"sql" jsonschema:"要执行的SQL查询语句"`
	DatabaseID string `json:"database_id" jsonschema:"数据库ID (数字)"`
}

type ExecuteSQLWithSchemaParams struct {
	SQL        string `json:"sql" jsonschema:"要执行的SQL查询语句"`
	DatabaseID string `json:"database_id" jsonschema:"数据库ID (数字)"`
	Schema     string `json:"schema" jsonschema:"数据库schema名称"`
}

type StatusParams struct{}

// createListDatabasesHandler 创建数据库列表处理器
func createListDatabasesHandler(client *Client) func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[ListDatabasesParams]) (*mcp.CallToolResultFor[any], error) {
	return func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[ListDatabasesParams]) (*mcp.CallToolResultFor[any], error) {
		if client == nil {
			return common.CreateErrorResponse("Superset客户端不可用")
		}

		databases, err := client.GetDatabases(ctx)
		if err != nil {
			return common.CreateErrorResponse("获取数据库列表失败: %v", err)
		}

		dbInfo := map[string]any{
			"count":     len(databases),
			"databases": databases,
		}

		return common.CreateSuccessResponse(dbInfo)
	}
}

// createExecuteSQLHandler 创建SQL执行处理器
func createExecuteSQLHandler(client *Client) func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[ExecuteSQLParams]) (*mcp.CallToolResultFor[any], error) {
	return func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[ExecuteSQLParams]) (*mcp.CallToolResultFor[any], error) {
		if client == nil {
			return common.CreateErrorResponse("Superset客户端不可用")
		}

		// 解析数据库ID
		databaseID, err := strconv.Atoi(params.Arguments.DatabaseID)
		if err != nil {
			return common.CreateErrorResponse("无效的数据库ID格式: %v", err)
		}

		result, err := client.ExecuteSQL(ctx, params.Arguments.SQL, databaseID)
		if err != nil {
			return common.CreateErrorResponse("执行SQL失败: %v", err)
		}

		return common.CreateSuccessResponse(result)
	}
}

// createExecuteSQLWithSchemaHandler 创建带schema的SQL执行处理器
func createExecuteSQLWithSchemaHandler(client *Client) func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[ExecuteSQLWithSchemaParams]) (*mcp.CallToolResultFor[any], error) {
	return func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[ExecuteSQLWithSchemaParams]) (*mcp.CallToolResultFor[any], error) {
		if client == nil {
			return common.CreateErrorResponse("Superset客户端不可用")
		}

		// 解析数据库ID
		databaseID, err := strconv.Atoi(params.Arguments.DatabaseID)
		if err != nil {
			return common.CreateErrorResponse("无效的数据库ID格式: %v", err)
		}

		result, err := client.ExecuteSQLWithSchema(ctx, params.Arguments.SQL, databaseID, params.Arguments.Schema)
		if err != nil {
			return common.CreateErrorResponse("执行SQL失败: %v", err)
		}

		return common.CreateSuccessResponse(result)
	}
}

// createStatusHandler 创建状态检查处理器
func createStatusHandler(client *Client) func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[StatusParams]) (*mcp.CallToolResultFor[any], error) {
	return func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[StatusParams]) (*mcp.CallToolResultFor[any], error) {
		if client == nil {
			return common.CreateErrorResponse("Superset客户端不可用")
		}

		// 测试连接
		if err := client.TestConnection(ctx); err != nil {
			return common.CreateErrorResponse("连接测试失败: %v", err)
		}

		// 尝试登录
		if err := client.Login(ctx); err != nil {
			return common.CreateErrorResponse("登录测试失败: %v", err)
		}

		// 尝试获取数据库列表来验证功能
		databases, err := client.GetDatabases(ctx)
		if err != nil {
			return common.CreateErrorResponse("功能测试失败: %v", err)
		}

		status := map[string]any{
			"status":     "connected",
			"message":    "Superset服务器连接正常",
			"login":      "success",
			"databases":  len(databases),
			"functional": "ready",
		}

		return common.CreateSuccessResponse(status)
	}
}
