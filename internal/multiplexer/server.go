package multiplexer

import (
	"bytes"
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"mcp-server/internal/core"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// 常量定义
const (
	// HTTP服务器配置
	readTimeout    = 30 * time.Second
	writeTimeout   = 30 * time.Second
	idleTimeout    = 120 * time.Second
	maxHeaderBytes = 1 << 20 // 1MB

	// 路由路径
	rootPath = "/"

	// HTTP响应
	contentTypeHTML   = "text/html; charset=utf-8"
	httpErrorInternal = "内部服务器错误"
)

// ServiceInfo 服务信息
type ServiceInfo struct {
	Type        core.ServiceType
	Endpoint    string
	Available   bool
	Tools       []string
	Description string
}

// Server HTTP多路复用服务器
type Server struct {
	services        map[string]core.Service // endpoint -> service 映射
	server          *http.Server
	port            string
	serverAddresses []string
	mu              sync.RWMutex

	// 网络地址缓存优化
	addressCache     []string
	addressCacheTime time.Time
	cacheMutex       sync.RWMutex
}

// NewServer 创建新的多路复用服务器
func NewServer(port string) *Server {
	server := &Server{
		services: make(map[string]core.Service),
		port:     port,
	}
	// 初始化时获取网络地址
	server.serverAddresses = server.getCachedServerAddresses()
	return server
}

// getCachedServerAddresses 获取缓存的服务器地址
func (s *Server) getCachedServerAddresses() []string {
	const cacheTimeout = 5 * time.Minute

	s.cacheMutex.RLock()
	if time.Since(s.addressCacheTime) < cacheTimeout && s.addressCache != nil {
		// 返回缓存副本，避免外部修改
		result := make([]string, len(s.addressCache))
		copy(result, s.addressCache)
		s.cacheMutex.RUnlock()
		return result
	}
	s.cacheMutex.RUnlock()

	// 双重检查锁模式，避免重复计算
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	if time.Since(s.addressCacheTime) < cacheTimeout && s.addressCache != nil {
		result := make([]string, len(s.addressCache))
		copy(result, s.addressCache)
		return result
	}

	// 重新获取网络地址并缓存
	addresses := getServerAddresses()
	s.addressCache = make([]string, len(addresses))
	copy(s.addressCache, addresses)
	s.addressCacheTime = time.Now()

	// 返回副本
	result := make([]string, len(addresses))
	copy(result, addresses)
	return result
}

// endpointFormatting 端点格式化
func endpointFormatting(addresses []string, port, endpoint string) string {
	if len(addresses) == 0 {
		return ""
	}

	var builder strings.Builder
	// 预估容量，减少重分配
	builder.Grow(len(addresses) * 50)

	for i, addr := range addresses {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString("http://")
		builder.WriteString(addr)
		builder.WriteString(":")
		builder.WriteString(port)
		builder.WriteString(endpoint)
	}
	return builder.String()
}

// AddService 添加服务
func (s *Server) AddService(service core.Service) {
	endpoint := service.GetEndpoint()
	serviceType := service.GetType()

	s.mu.Lock()
	s.services[endpoint] = service
	s.mu.Unlock()

	log.Printf("✓ 注册服务: %s -> %s", serviceType, endpoint)
}

// RemoveService 移除服务
func (s *Server) RemoveService(endpoint string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if service, exists := s.services[endpoint]; exists {
		service.Close()
		delete(s.services, endpoint)
		log.Printf("移除服务: %s", endpoint)
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	mux := http.NewServeMux()

	s.mu.RLock()
	servicesCopy := make(map[string]core.Service, len(s.services))
	for k, v := range s.services {
		servicesCopy[k] = v
	}
	s.mu.RUnlock()

	for endpoint, service := range servicesCopy {
		// 创建服务处理器
		handler := mcp.NewStreamableHTTPHandler(
			func(request *http.Request) *mcp.Server {
				return service.GetServer()
			},
			&mcp.StreamableHTTPOptions{},
		)
		mux.Handle(endpoint, handler)

		// 使用字符串格式化
		endpointsStr := endpointFormatting(s.serverAddresses, s.port, endpoint)
		log.Printf("%s MCP端点: %s", service.GetType(), endpointsStr)
	}

	// 添加根路径信息页面
	mux.HandleFunc(rootPath, s.handleRoot)

	serverAddrsStr := endpointFormatting(s.serverAddresses, s.port, "")
	log.Printf("服务器监听地址: %s", serverAddrsStr)

	// 创建HTTP服务器
	s.server = &http.Server{
		Addr:           ":" + s.port,
		Handler:        mux,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		IdleTimeout:    idleTimeout,
		MaxHeaderBytes: maxHeaderBytes,
	}

	return s.server.ListenAndServe()
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	servicesCopy := make([]core.Service, 0, len(s.services))
	for _, service := range s.services {
		servicesCopy = append(servicesCopy, service)
	}
	s.mu.RUnlock()

	for _, service := range servicesCopy {
		service.Close()
	}

	// 关闭HTTP服务器
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// GetServiceInfo 获取服务信息
func (s *Server) GetServiceInfo() []ServiceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infos := make([]ServiceInfo, 0, len(s.services))
	for endpoint, service := range s.services {
		info := ServiceInfo{
			Type:        service.GetType(),
			Endpoint:    endpoint,
			Available:   true, // 已注册的服务都是可用的
			Tools:       getToolsForService(service.GetType()),
			Description: getDescriptionForService(service.GetType()),
		}
		infos = append(infos, info)
	}

	return infos
}

// handleRoot 处理根路径请求
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != rootPath {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", contentTypeHTML)

	// 准备模板数据
	serviceInfos := s.GetServiceInfo()
	data := struct {
		ServerAddresses []string
		Port            string
		Services        []ServiceInfo
	}{
		ServerAddresses: s.serverAddresses,
		Port:            s.port,
		Services:        serviceInfos,
	}

	// 使用缓冲区来提高性能
	var buf bytes.Buffer
	if err := htmlTemplate.Execute(&buf, data); err != nil {
		log.Printf("模板执行错误: %v", err)
		http.Error(w, httpErrorInternal, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("写入响应错误: %v", err)
	}
}

// getServerAddresses 获取服务器地址列表
func getServerAddresses() []string {
	addressSet := make(map[string]bool)
	addresses := make([]string, 0, 4)

	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("警告: 无法获取网络接口: %v", err)
		return []string{"localhost"}
	}

	for _, iface := range interfaces {
		// 跳过回环接口和down状态的接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			log.Printf("警告: 无法获取接口 %s 的地址: %v", iface.Name, err)
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				// 只获取IPv4地址，排除特殊地址
				if ip4 := ipnet.IP.To4(); ip4 != nil && !ip4.IsLoopback() {
					if !isDockerOrVirtualIP(ip4) {
						ipStr := ip4.String()
						if !addressSet[ipStr] {
							addressSet[ipStr] = true
							addresses = append(addresses, ipStr)
						}
					}
				}
			}
		}
	}

	// 如果没有找到任何地址，使用localhost
	if len(addresses) == 0 {
		addresses = append(addresses, "localhost")
	}

	return addresses
}

// isDockerOrVirtualIP 检查是否为Docker或其他虚拟网络的IP
func isDockerOrVirtualIP(ip net.IP) bool {
	// Docker默认网络: 172.17.0.0/16
	dockerNet := &net.IPNet{
		IP:   net.IPv4(172, 17, 0, 0),
		Mask: net.CIDRMask(16, 32),
	}

	// Docker用户定义网络: 172.18.0.0/16 - 172.31.0.0/16
	for i := 18; i <= 31; i++ {
		dockerUserNet := &net.IPNet{
			IP:   net.IPv4(172, byte(i), 0, 0),
			Mask: net.CIDRMask(16, 32),
		}
		if dockerUserNet.Contains(ip) {
			return true
		}
	}

	return dockerNet.Contains(ip)
}

// getToolsForService 获取服务的工具列表
func getToolsForService(serviceType core.ServiceType) []string {
	switch serviceType {
	case core.ServiceTypePrometheus:
		return []string{
			"prometheus_query - 执行即时查询",
			"prometheus_query_range - 执行范围查询",
			"prometheus_targets - 获取监控目标",
			"prometheus_status - 检查服务状态",
			"prometheus_common_metrics - 查询常用指标",
			"prometheus_list_metrics - 获取所有指标",
		}
	case core.ServiceTypeSuperset:
		return []string{
			"superset_list_databases - 获取数据库列表",
			"superset_execute_sql - 执行SQL查询",
			"superset_execute_sql_with_schema - 在指定schema中执行SQL",
			"superset_status - 检查服务状态",
		}
	default:
		return []string{}
	}
}

// getDescriptionForService 获取服务描述
func getDescriptionForService(serviceType core.ServiceType) string {
	switch serviceType {
	case core.ServiceTypePrometheus:
		return "提供Prometheus监控数据查询功能"
	case core.ServiceTypeSuperset:
		return "提供Superset数据库查询和管理功能"
	default:
		return "MCP服务"
	}
}
