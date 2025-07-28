package services

import (
	"mcp-server/internal/core"
	"mcp-server/internal/services/prometheus"
	"mcp-server/internal/services/superset"
)

// 注册服务
func init() {
	core.RegisterServiceFactory(core.ServiceTypePrometheus, prometheus.CreateService)
	core.RegisterServiceFactory(core.ServiceTypeSuperset, superset.CreateService)
}
