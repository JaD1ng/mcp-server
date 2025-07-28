package superset

import (
	"context"
	"fmt"
	"time"

	"mcp-server/config"
	"mcp-server/internal/core"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serviceImpl Superset服务实现
type serviceImpl struct {
	client   *Client
	server   *mcp.Server
	endpoint string
}

// CreateService 创建Superset服务实例（工厂函数）
func CreateService(serviceConfig core.ServiceConfig, timeout time.Duration) (core.Service, error) {
	supersetConfig, ok := serviceConfig.(*config.SupersetConfig)
	if !ok {
		return nil, fmt.Errorf("配置类型错误: 期望SupersetConfig，得到%T", serviceConfig)
	}

	// 创建客户端
	client, err := NewClient(supersetConfig.URL, supersetConfig.User, supersetConfig.Pass, timeout)
	if err != nil {
		return nil, core.NewServiceCreationError(core.ServiceTypeSuperset, err)
	}

	// 创建MCP服务器
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "Superset MCP Server",
		Version: "1.0.0",
	}, nil)

	service := &serviceImpl{
		client:   client,
		server:   server,
		endpoint: supersetConfig.GetEndpoint(),
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
	// Superset客户端无需特殊清理
	return nil
}

// GetType 实现Service接口
func (s *serviceImpl) GetType() core.ServiceType {
	return core.ServiceTypeSuperset
}

// GetEndpoint 实现Service接口
func (s *serviceImpl) GetEndpoint() string {
	return s.endpoint
}

// registerTools 注册所有Superset工具
func registerTools(server *mcp.Server, client *Client) {
	// 注册数据库列表工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "superset_list_databases",
		Description: "获取所有可用的数据库列表",
	}, createListDatabasesHandler(client))

	// 注册SQL执行工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "superset_execute_sql",
		Description: "在指定数据库中执行SQL查询",
	}, createExecuteSQLHandler(client))

	// 注册带schema的SQL执行工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "superset_execute_sql_with_schema",
		Description: "在指定数据库和schema中执行SQL查询",
	}, createExecuteSQLWithSchemaHandler(client))

	// 注册状态检查工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "superset_status",
		Description: "检查Superset服务状态和连接",
	}, createStatusHandler(client))
}
