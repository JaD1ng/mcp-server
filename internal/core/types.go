package core

import (
	"context"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServiceType MCP服务类型
type ServiceType string

const (
	ServiceTypePrometheus ServiceType = "prometheus"
	ServiceTypeSuperset   ServiceType = "superset"
)

// ServiceConfig 服务配置接口
type ServiceConfig interface {
	GetType() ServiceType
	GetEndpoint() string
	IsEnabled() bool
	Validate() error
}

// ServiceFactory 服务工厂函数类型
type ServiceFactory func(config ServiceConfig, timeout time.Duration) (Service, error)

// Service MCP服务接口
type Service interface {
	// GetServer 获取MCP服务器实例
	GetServer() *mcp.Server

	// TestConnection 测试服务连接
	TestConnection(ctx context.Context) error

	// Close 关闭服务连接
	Close() error

	// GetType 获取服务类型
	GetType() ServiceType

	// GetEndpoint 获取服务端点路径
	GetEndpoint() string
}

// 函数式Registry设计 - 使用全局不可变映射
var serviceFactories = make(map[ServiceType]ServiceFactory)
var factoriesMutex sync.RWMutex
var supportedTypesCache []ServiceType
var cacheValid bool

// RegisterServiceFactory 注册服务工厂函数
func RegisterServiceFactory(serviceType ServiceType, factory ServiceFactory) {
	factoriesMutex.Lock()
	defer factoriesMutex.Unlock()
	serviceFactories[serviceType] = factory
	// 使缓存失效
	cacheValid = false
}

// CreateService 创建服务实例
func CreateService(config ServiceConfig, timeout time.Duration) (Service, error) {
	factoriesMutex.RLock()
	factory, exists := serviceFactories[config.GetType()]
	factoriesMutex.RUnlock()

	if !exists {
		return nil, NewUnsupportedServiceError(config.GetType())
	}

	return factory(config, timeout)
}

// GetSupportedServiceTypes 获取支持的服务类型
func GetSupportedServiceTypes() []ServiceType {
	factoriesMutex.RLock()

	// 检查缓存是否有效
	if cacheValid && supportedTypesCache != nil {
		// 返回缓存的副本以避免外部修改
		result := make([]ServiceType, len(supportedTypesCache))
		copy(result, supportedTypesCache)
		factoriesMutex.RUnlock()
		return result
	}

	// 重建缓存
	types := make([]ServiceType, 0, len(serviceFactories))
	for serviceType := range serviceFactories {
		types = append(types, serviceType)
	}

	// 更新缓存
	supportedTypesCache = make([]ServiceType, len(types))
	copy(supportedTypesCache, types)
	cacheValid = true

	factoriesMutex.RUnlock()
	return types
}

// IsServiceTypeSupported 检查服务类型是否支持
func IsServiceTypeSupported(serviceType ServiceType) bool {
	factoriesMutex.RLock()
	defer factoriesMutex.RUnlock()
	_, exists := serviceFactories[serviceType]
	return exists
}

// Legacy Registry 结构体
type Registry struct{}

// NewRegistry 创建新的服务注册表
func NewRegistry() *Registry {
	return &Registry{}
}

// Register 注册服务工厂
func (r *Registry) Register(serviceType ServiceType, factory ServiceFactory) {
	RegisterServiceFactory(serviceType, factory)
}

// Create 创建服务实例
func (r *Registry) Create(config ServiceConfig, timeout time.Duration) (Service, error) {
	return CreateService(config, timeout)
}

// GetSupportedTypes 获取支持的服务类型
func (r *Registry) GetSupportedTypes() []ServiceType {
	return GetSupportedServiceTypes()
}
