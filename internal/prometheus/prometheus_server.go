package prometheus

import (
	"context"
	"time"

	"mcp-server/internal/common"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// 常量定义
const (
	// 错误消息
	errClientUnavailable    = "Prometheus客户端不可用"
	errQueryFailed          = "查询失败: %v"
	errRangeQueryFailed     = "范围查询失败: %v"
	errGetTargetsFailed     = "获取目标失败: %v"
	errConnectionTestFailed = "连接测试失败: %v"
	errFunctionalTestFailed = "功能测试失败: %v"
	errUnsupportedMetric    = "不支持的指标类型"
	errGetMetricsFailed     = "获取指标名称失败: %v"
	errJSONMarshalFailed    = "结果转换失败: %v"
	errInvalidTimeFormat    = "无效的%s时间格式: %v"
	errInvalidStepFormat    = "无效的步长格式: %v"

	// 状态消息
	statusConnected = "connected"
	statusMessage   = "Prometheus服务器连接正常"

	// 超时配置
	defaultQueryTimeout = 10 * time.Second
	rangeQueryTimeout   = 30 * time.Second
	listMetricsTimeout  = 15 * time.Second
)

// PrometheusMCPServer Prometheus专用的MCP服务器
type PrometheusMCPServer struct {
	client *Client
	server *mcp.Server
}

// NewPrometheusMCPServer 创建Prometheus MCP服务器实例
func NewPrometheusMCPServer(client *Client) *PrometheusMCPServer {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "Prometheus MCP Server",
		Version: "1.0.0",
	}, nil)

	prometheusServer := &PrometheusMCPServer{
		client: client,
		server: server,
	}

	// 注册Prometheus工具
	prometheusServer.registerTools()

	return prometheusServer
}

// createSuccessResponse 创建成功响应结果
func (pms *PrometheusMCPServer) createSuccessResponse(data any) (*mcp.CallToolResultFor[any], error) {
	return common.CreateSuccessResponse(data)
}

// createErrorResponse 创建错误响应结果
func (pms *PrometheusMCPServer) createErrorResponse(format string, args ...any) (*mcp.CallToolResultFor[any], error) {
	return common.CreateErrorResponse(format, args...)
}

// checkClientAvailability 检查客户端可用性状态
func (pms *PrometheusMCPServer) checkClientAvailability() (*mcp.CallToolResultFor[any], bool) {
	if pms.client == nil {
		return &mcp.CallToolResultFor[any]{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: errClientUnavailable}},
		}, false
	}
	return nil, true
}

// registerTools 注册所有Prometheus工具到MCP服务器
func (pms *PrometheusMCPServer) registerTools() {
	// 注册即时查询工具
	mcp.AddTool(pms.server, &mcp.Tool{
		Name:        "prometheus_query",
		Description: "执行Prometheus即时查询",
	}, pms.handleQuery)

	// 注册范围查询工具
	mcp.AddTool(pms.server, &mcp.Tool{
		Name:        "prometheus_query_range",
		Description: "执行Prometheus范围查询",
	}, pms.handleQueryRange)

	// 注册目标获取工具
	mcp.AddTool(pms.server, &mcp.Tool{
		Name:        "prometheus_targets",
		Description: "获取Prometheus监控目标",
	}, pms.handleTargets)

	// 注册状态检查工具
	mcp.AddTool(pms.server, &mcp.Tool{
		Name:        "prometheus_status",
		Description: "检查Prometheus服务状态和连接",
	}, pms.handleStatus)

	// 注册常用指标查询工具
	mcp.AddTool(pms.server, &mcp.Tool{
		Name:        "prometheus_common_metrics",
		Description: "查询常用Prometheus指标",
	}, pms.handleCommonMetrics)

	// 注册指标列表工具
	mcp.AddTool(pms.server, &mcp.Tool{
		Name:        "prometheus_list_metrics",
		Description: "获取所有可用的指标名称",
	}, pms.handleListMetrics)
}

// 工具参数结构体
type QueryParams struct {
	Query string `json:"query" jsonschema:"PromQL查询语句"`
}

type QueryRangeParams struct {
	Query     string `json:"query" jsonschema:"PromQL查询语句"`
	StartTime string `json:"start_time" jsonschema:"开始时间 (RFC3339格式, 例如: 2024-01-01T00:00:00Z)"`
	EndTime   string `json:"end_time" jsonschema:"结束时间 (RFC3339格式, 例如: 2024-01-01T23:59:59Z)"`
	Step      string `json:"step" jsonschema:"步长持续时间 (例如: 1m, 5m, 1h)"`
}

type TargetsParams struct{}

type StatusParams struct{}

type CommonMetricsParams struct {
	MetricType string `json:"metric_type" jsonschema:"指标类型 (cpu, memory, disk, network, up)"`
}

type ListMetricsParams struct{}

// handleQuery 处理Prometheus即时查询请求
func (pms *PrometheusMCPServer) handleQuery(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[QueryParams]) (*mcp.CallToolResultFor[any], error) {
	if errResp, ok := pms.checkClientAvailability(); !ok {
		return errResp, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	result, err := pms.client.QueryInstant(queryCtx, params.Arguments.Query)
	if err != nil {
		return pms.createErrorResponse(errQueryFailed, err)
	}

	return pms.createSuccessResponse(result)
}

// handleQueryRange 处理Prometheus范围查询请求
func (pms *PrometheusMCPServer) handleQueryRange(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[QueryRangeParams]) (*mcp.CallToolResultFor[any], error) {
	if errResp, ok := pms.checkClientAvailability(); !ok {
		return errResp, nil
	}

	// 预先验证所有时间参数
	startTime, err := time.Parse(time.RFC3339, params.Arguments.StartTime)
	if err != nil {
		return pms.createErrorResponse(errInvalidTimeFormat, "开始", err)
	}

	endTime, err := time.Parse(time.RFC3339, params.Arguments.EndTime)
	if err != nil {
		return pms.createErrorResponse(errInvalidTimeFormat, "结束", err)
	}

	step, err := time.ParseDuration(params.Arguments.Step)
	if err != nil {
		return pms.createErrorResponse(errInvalidStepFormat, err)
	}

	queryCtx, cancel := context.WithTimeout(ctx, rangeQueryTimeout)
	defer cancel()

	result, err := pms.client.QueryRange(queryCtx, params.Arguments.Query, startTime, endTime, step)
	if err != nil {
		return pms.createErrorResponse(errRangeQueryFailed, err)
	}

	return pms.createSuccessResponse(result)
}

// handleTargets 处理获取Prometheus监控目标请求
func (pms *PrometheusMCPServer) handleTargets(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[TargetsParams]) (*mcp.CallToolResultFor[any], error) {
	if errResp, ok := pms.checkClientAvailability(); !ok {
		return errResp, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	targets, err := pms.client.GetTargets(queryCtx)
	if err != nil {
		return pms.createErrorResponse(errGetTargetsFailed, err)
	}

	targetInfo := map[string]any{
		"active_count":  len(targets.Active),
		"dropped_count": len(targets.Dropped),
		"active":        targets.Active,
		"dropped":       targets.Dropped,
	}

	return pms.createSuccessResponse(targetInfo)
}

// handleStatus 处理Prometheus服务状态检查请求
func (pms *PrometheusMCPServer) handleStatus(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[StatusParams]) (*mcp.CallToolResultFor[any], error) {
	if errResp, ok := pms.checkClientAvailability(); !ok {
		return errResp, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	err := pms.client.TestConnection(queryCtx)
	if err != nil {
		return pms.createErrorResponse(errConnectionTestFailed, err)
	}

	result, err := pms.client.QueryInstant(queryCtx, "up")
	if err != nil {
		return pms.createErrorResponse(errFunctionalTestFailed, err)
	}

	status := map[string]any{
		"status":    statusConnected,
		"message":   statusMessage,
		"up_result": result,
	}

	return pms.createSuccessResponse(status)
}

// handleCommonMetrics 处理常用Prometheus指标查询请求
func (pms *PrometheusMCPServer) handleCommonMetrics(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[CommonMetricsParams]) (*mcp.CallToolResultFor[any], error) {
	if errResp, ok := pms.checkClientAvailability(); !ok {
		return errResp, nil
	}

	query, exists := MetricQueries[params.Arguments.MetricType]
	if !exists {
		return pms.createErrorResponse(errUnsupportedMetric)
	}

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	result, err := pms.client.QueryInstant(queryCtx, query)
	if err != nil {
		return pms.createErrorResponse(errQueryFailed, err)
	}

	return pms.createSuccessResponse(result)
}

// handleListMetrics 处理获取所有可用指标名称请求
func (pms *PrometheusMCPServer) handleListMetrics(ctx context.Context, cc *mcp.ServerSession, params *mcp.CallToolParamsFor[ListMetricsParams]) (*mcp.CallToolResultFor[any], error) {
	if errResp, ok := pms.checkClientAvailability(); !ok {
		return errResp, nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, listMetricsTimeout)
	defer cancel()

	metricNames, err := pms.client.GetMetricNames(queryCtx)
	if err != nil {
		return pms.createErrorResponse(errGetMetricsFailed, err)
	}

	result := map[string]any{
		"count":   len(metricNames),
		"metrics": metricNames,
	}

	return pms.createSuccessResponse(result)
}

// GetServer 获取MCP服务器实例
func (pms *PrometheusMCPServer) GetServer() *mcp.Server {
	return pms.server
}
