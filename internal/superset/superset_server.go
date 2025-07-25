package superset

import (
	"context"
	"strconv"

	"mcp-server/internal/common"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// 常量定义
const (
	// 错误消息
	errClientUnavailable    = "Superset客户端不可用"
	errInvalidDatabaseID    = "无效的数据库ID格式: %v"
	errGetDatabasesFailed   = "获取数据库列表失败: %v"
	errExecuteSQLFailed     = "执行SQL失败: %v"
	errConnectionTestFailed = "连接测试失败: %v"
	errLoginTestFailed      = "登录测试失败: %v"
	errFunctionalTestFailed = "功能测试失败: %v"
	errJSONMarshalFailed    = "结果转换失败: %v"

	// 状态消息
	statusConnected       = "connected"
	statusMessage         = "Superset服务器连接正常"
	statusLoginSuccess    = "success"
	statusFunctionalReady = "ready"
)

// SupersetMCPServer Superset专用的MCP服务器
type SupersetMCPServer struct {
	client *Client
	server *mcp.Server
}

// NewSupersetMCPServer 创建Superset MCP服务器
func NewSupersetMCPServer(client *Client) *SupersetMCPServer {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "Superset MCP Server",
		Version: "1.0.0",
	}, nil)

	supersetServer := &SupersetMCPServer{
		client: client,
		server: server,
	}

	// 注册Superset工具
	supersetServer.registerTools()

	return supersetServer
}

// registerTools 注册Superset工具
func (sms *SupersetMCPServer) registerTools() {
	// 注册数据库列表工具
	mcp.AddTool(sms.server, &mcp.Tool{
		Name:        "superset_list_databases",
		Description: "获取所有可用的数据库列表",
	}, sms.handleListDatabases)

	// 注册SQL执行工具
	mcp.AddTool(sms.server, &mcp.Tool{
		Name:        "superset_execute_sql",
		Description: "在指定数据库中执行SQL查询",
	}, sms.handleExecuteSQL)

	// 注册带schema的SQL执行工具
	mcp.AddTool(sms.server, &mcp.Tool{
		Name:        "superset_execute_sql_with_schema",
		Description: "在指定数据库和schema中执行SQL查询",
	}, sms.handleExecuteSQLWithSchema)

	// 注册状态检查工具
	mcp.AddTool(sms.server, &mcp.Tool{
		Name:        "superset_status",
		Description: "检查Superset服务状态和连接",
	}, sms.handleStatus)
}

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

// 工具处理器
func (sms *SupersetMCPServer) handleListDatabases(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ListDatabasesParams]) (*mcp.CallToolResultFor[any], error) {
	// 验证客户端
	if resp, err, ok := sms.validateClient(); !ok {
		return resp, err
	}

	databases, err := sms.client.GetDatabases(ctx)
	if err != nil {
		return createErrorResponse(errGetDatabasesFailed, err)
	}

	dbInfo := map[string]any{
		"count":     len(databases),
		"databases": databases,
	}

	return createJSONResponse(dbInfo)
}

func (sms *SupersetMCPServer) handleExecuteSQL(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ExecuteSQLParams]) (*mcp.CallToolResultFor[any], error) {
	// 验证客户端
	if resp, err, ok := sms.validateClient(); !ok {
		return resp, err
	}

	// 解析数据库ID
	databaseID, resp, err, ok := parseDatabaseID(params.Arguments.DatabaseID)
	if !ok {
		return resp, err
	}

	result, err := sms.client.ExecuteSQL(ctx, params.Arguments.SQL, databaseID)
	if err != nil {
		return createErrorResponse(errExecuteSQLFailed, err)
	}

	return createJSONResponse(result)
}

func (sms *SupersetMCPServer) handleExecuteSQLWithSchema(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ExecuteSQLWithSchemaParams]) (*mcp.CallToolResultFor[any], error) {
	// 验证客户端
	if resp, err, ok := sms.validateClient(); !ok {
		return resp, err
	}

	// 解析数据库ID
	databaseID, resp, err, ok := parseDatabaseID(params.Arguments.DatabaseID)
	if !ok {
		return resp, err
	}

	result, err := sms.client.ExecuteSQLWithSchema(ctx, params.Arguments.SQL, databaseID, params.Arguments.Schema)
	if err != nil {
		return createErrorResponse(errExecuteSQLFailed, err)
	}

	return createJSONResponse(result)
}

func (sms *SupersetMCPServer) handleStatus(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[StatusParams]) (*mcp.CallToolResultFor[any], error) {
	// 验证客户端
	if resp, err, ok := sms.validateClient(); !ok {
		return resp, err
	}

	// 测试连接
	if err := sms.client.TestConnection(ctx); err != nil {
		return createErrorResponse(errConnectionTestFailed, err)
	}

	// 尝试登录
	if err := sms.client.Login(ctx); err != nil {
		return createErrorResponse(errLoginTestFailed, err)
	}

	// 尝试获取数据库列表来验证功能
	databases, err := sms.client.GetDatabases(ctx)
	if err != nil {
		return createErrorResponse(errFunctionalTestFailed, err)
	}

	status := map[string]any{
		"status":     statusConnected,
		"message":    statusMessage,
		"login":      statusLoginSuccess,
		"databases":  len(databases),
		"functional": statusFunctionalReady,
	}

	return createJSONResponse(status)
}

// GetServer 获取MCP服务器实例
func (sms *SupersetMCPServer) GetServer() *mcp.Server {
	return sms.server
}

// 辅助函数 - 创建错误响应
func createErrorResponse(format string, args ...any) (*mcp.CallToolResultFor[any], error) {
	return common.CreateErrorResponse(format, args...)
}

// 辅助函数 - 创建JSON响应
func createJSONResponse(data any) (*mcp.CallToolResultFor[any], error) {
	return common.CreateSuccessResponse(data)
}

// 辅助函数 - 验证客户端可用性
func (sms *SupersetMCPServer) validateClient() (*mcp.CallToolResultFor[any], error, bool) {
	if sms.client == nil {
		resp, err := createErrorResponse(errClientUnavailable)
		return resp, err, false
	}
	return nil, nil, true
}

// 辅助函数 - 解析数据库ID
func parseDatabaseID(id string) (int, *mcp.CallToolResultFor[any], error, bool) {
	databaseID, err := strconv.Atoi(id)
	if err != nil {
		resp, err := createErrorResponse(errInvalidDatabaseID, err)
		return 0, resp, err, false
	}
	return databaseID, nil, nil, true
}
