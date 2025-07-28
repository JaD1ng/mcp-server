package prometheus

import (
	"context"
	"fmt"
	"time"

	"mcp-server/config"
	"mcp-server/internal/core"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serviceImpl Prometheus服务实现
type serviceImpl struct {
	client   *Client
	server   *mcp.Server
	endpoint string
}

// CreateService 创建Prometheus服务实例（工厂函数）
func CreateService(serviceConfig core.ServiceConfig, timeout time.Duration) (core.Service, error) {
	promConfig, ok := serviceConfig.(*config.PrometheusConfig)
	if !ok {
		return nil, fmt.Errorf("配置类型错误: 期望PrometheusConfig，得到%T", serviceConfig)
	}

	// 创建客户端
	client, err := NewClient(promConfig.URL)
	if err != nil {
		return nil, core.NewServiceCreationError(core.ServiceTypePrometheus, err)
	}

	// 创建MCP服务器
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "Prometheus MCP Server",
		Version: "1.0.0",
	}, nil)

	service := &serviceImpl{
		client:   client,
		server:   server,
		endpoint: promConfig.GetEndpoint(),
	}

	// 注册工具
	registerTools(server, client)

	return service, nil
}

// GetServer 实现Service接口
func (s *serviceImpl) GetServer() *mcp.Server {
	return s.server
}

// TestConnection 实现Service接口
func (s *serviceImpl) TestConnection(ctx context.Context) error {
	if s.client == nil {
		return fmt.Errorf("客户端未初始化")
	}
	return s.client.TestConnection(ctx)
}

// Close 实现Service接口
func (s *serviceImpl) Close() error {
	// Prometheus客户端无需特殊清理
	return nil
}

// GetType 实现Service接口
func (s *serviceImpl) GetType() core.ServiceType {
	return core.ServiceTypePrometheus
}

// GetEndpoint 实现Service接口
func (s *serviceImpl) GetEndpoint() string {
	return s.endpoint
}

// registerTools 注册所有Prometheus工具
func registerTools(server *mcp.Server, client *Client) {
	// 注册即时查询工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "prometheus_query",
		Description: "执行Prometheus即时查询",
	}, createQueryHandler(client))

	// 注册范围查询工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "prometheus_query_range",
		Description: "执行Prometheus范围查询",
	}, createQueryRangeHandler(client))

	// 注册目标获取工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "prometheus_targets",
		Description: "获取Prometheus监控目标",
	}, createTargetsHandler(client))

	// 注册状态检查工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "prometheus_status",
		Description: "检查Prometheus服务状态和连接",
	}, createStatusHandler(client))

	// 注册常用指标查询工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "prometheus_common_metrics",
		Description: "查询常用Prometheus指标",
	}, createCommonMetricsHandler(client))

	// 注册指标列表工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "prometheus_list_metrics",
		Description: "获取所有可用的指标名称",
	}, createListMetricsHandler(client))
}
