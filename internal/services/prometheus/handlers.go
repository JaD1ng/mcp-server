package prometheus

import (
	"context"
	"time"

	"mcp-server/internal/common"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// 常量定义
const (
	defaultQueryTimeout = 10 * time.Second
	rangeQueryTimeout   = 30 * time.Second
	listMetricsTimeout  = 15 * time.Second
)

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

// createQueryHandler 创建即时查询处理器
func createQueryHandler(client *Client) func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[QueryParams]) (*mcp.CallToolResultFor[any], error) {
	return func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[QueryParams]) (*mcp.CallToolResultFor[any], error) {
		if client == nil {
			return common.CreateErrorResponse("Prometheus客户端不可用")
		}

		queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancel()

		result, err := client.QueryInstant(queryCtx, params.Arguments.Query)
		if err != nil {
			return common.CreateErrorResponse("查询失败: %v", err)
		}

		return common.CreateSuccessResponse(result)
	}
}

// createQueryRangeHandler 创建范围查询处理器
func createQueryRangeHandler(client *Client) func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[QueryRangeParams]) (*mcp.CallToolResultFor[any], error) {
	return func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[QueryRangeParams]) (*mcp.CallToolResultFor[any], error) {
		if client == nil {
			return common.CreateErrorResponse("Prometheus客户端不可用")
		}

		// 验证时间参数
		startTime, err := time.Parse(time.RFC3339, params.Arguments.StartTime)
		if err != nil {
			return common.CreateErrorResponse("无效的开始时间格式: %v", err)
		}

		endTime, err := time.Parse(time.RFC3339, params.Arguments.EndTime)
		if err != nil {
			return common.CreateErrorResponse("无效的结束时间格式: %v", err)
		}

		step, err := time.ParseDuration(params.Arguments.Step)
		if err != nil {
			return common.CreateErrorResponse("无效的步长格式: %v", err)
		}

		queryCtx, cancel := context.WithTimeout(ctx, rangeQueryTimeout)
		defer cancel()

		result, err := client.QueryRange(queryCtx, params.Arguments.Query, startTime, endTime, step)
		if err != nil {
			return common.CreateErrorResponse("范围查询失败: %v", err)
		}

		return common.CreateSuccessResponse(result)
	}
}

// createTargetsHandler 创建目标获取处理器
func createTargetsHandler(client *Client) func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[TargetsParams]) (*mcp.CallToolResultFor[any], error) {
	return func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[TargetsParams]) (*mcp.CallToolResultFor[any], error) {
		if client == nil {
			return common.CreateErrorResponse("Prometheus客户端不可用")
		}

		queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancel()

		targets, err := client.GetTargets(queryCtx)
		if err != nil {
			return common.CreateErrorResponse("获取目标失败: %v", err)
		}

		targetInfo := map[string]any{
			"active_count":  len(targets.Active),
			"dropped_count": len(targets.Dropped),
			"active":        targets.Active,
			"dropped":       targets.Dropped,
		}

		return common.CreateSuccessResponse(targetInfo)
	}
}

// createStatusHandler 创建状态检查处理器
func createStatusHandler(client *Client) func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[StatusParams]) (*mcp.CallToolResultFor[any], error) {
	return func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[StatusParams]) (*mcp.CallToolResultFor[any], error) {
		if client == nil {
			return common.CreateErrorResponse("Prometheus客户端不可用")
		}

		queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancel()

		// 测试连接
		if err := client.TestConnection(queryCtx); err != nil {
			return common.CreateErrorResponse("连接测试失败: %v", err)
		}

		// 功能测试
		result, err := client.QueryInstant(queryCtx, "up")
		if err != nil {
			return common.CreateErrorResponse("功能测试失败: %v", err)
		}

		status := map[string]any{
			"status":    "connected",
			"message":   "Prometheus服务器连接正常",
			"up_result": result,
		}

		return common.CreateSuccessResponse(status)
	}
}

// createCommonMetricsHandler 创建常用指标查询处理器
func createCommonMetricsHandler(client *Client) func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[CommonMetricsParams]) (*mcp.CallToolResultFor[any], error) {
	return func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[CommonMetricsParams]) (*mcp.CallToolResultFor[any], error) {
		if client == nil {
			return common.CreateErrorResponse("Prometheus客户端不可用")
		}

		query, exists := MetricQueries[params.Arguments.MetricType]
		if !exists {
			return common.CreateErrorResponse("不支持的指标类型")
		}

		queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
		defer cancel()

		result, err := client.QueryInstant(queryCtx, query)
		if err != nil {
			return common.CreateErrorResponse("查询失败: %v", err)
		}

		return common.CreateSuccessResponse(result)
	}
}

// createListMetricsHandler 创建指标列表处理器
func createListMetricsHandler(client *Client) func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[ListMetricsParams]) (*mcp.CallToolResultFor[any], error) {
	return func(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[ListMetricsParams]) (*mcp.CallToolResultFor[any], error) {
		if client == nil {
			return common.CreateErrorResponse("Prometheus客户端不可用")
		}

		queryCtx, cancel := context.WithTimeout(ctx, listMetricsTimeout)
		defer cancel()

		metricNames, err := client.GetMetricNames(queryCtx)
		if err != nil {
			return common.CreateErrorResponse("获取指标名称失败: %v", err)
		}

		result := map[string]any{
			"count":   len(metricNames),
			"metrics": metricNames,
		}

		return common.CreateSuccessResponse(result)
	}
}
