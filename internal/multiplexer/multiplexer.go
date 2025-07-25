package multiplexer

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

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
	supersetMCPPath   = "/superset/mcp"
	prometheusMCPPath = "/prometheus/mcp"
	rootPath          = "/"

	// 日志消息
	logServerAddresses    = "服务器监听地址: %s"
	logSupersetEndpoint   = "Superset MCP端点: http://%s:%s/superset/mcp"
	logPrometheusEndpoint = "Prometheus MCP端点: http://%s:%s/prometheus/mcp"
	logTemplateError      = "模板执行错误: %v"
	logWriteError         = "写入响应错误: %v"
	logInterfaceError     = "警告: 无法获取网络接口: %v"
	logAddressError       = "警告: 无法获取接口 %s 的地址: %v"

	// HTTP响应
	contentTypeHTML   = "text/html; charset=utf-8"
	httpErrorInternal = "内部服务器错误"
)

// htmlTemplate 预编译的HTML模板
var htmlTemplate = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
<head>
    <title>MCP服务器</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { background: #f5f5f5; padding: 20px; margin: 20px 0; border-radius: 5px; }
        .endpoint h3 { margin-top: 0; color: #333; }
        .endpoint a { color: #007bff; text-decoration: none; }
        .endpoint a:hover { text-decoration: underline; }
        .status { padding: 5px 10px; border-radius: 3px; font-size: 12px; font-weight: bold; }
        .status.available { background: #d4edda; color: #155724; }
        .status.unavailable { background: #f8d7da; color: #721c24; }
        .server-addresses { background: #e9ecef; padding: 10px; margin: 10px 0; border-radius: 3px; font-family: monospace; }
    </style>
</head>
<body>
    <h1>MCP服务器</h1>
    <p>欢迎使用MCP服务器。以下是可用的端点：</p>
    
    <div class="server-addresses">
        <strong>服务器地址:</strong><br>
        {{range .ServerAddresses}}• http://{{.}}:{{$.Port}}<br>{{end}}
    </div>

    {{if .SupersetAvailable}}
    <div class="endpoint">
        <h3>Superset MCP服务器 <span class="status available">可用</span></h3>
        <p>提供Superset数据库查询和管理功能</p>
        <p><strong>端点:</strong> <a href="/superset/mcp">/superset/mcp</a></p>
        <p><strong>可用工具:</strong></p>
        <ul>
            <li>superset_list_databases - 获取数据库列表</li>
            <li>superset_execute_sql - 执行SQL查询</li>
            <li>superset_execute_sql_with_schema - 在指定schema中执行SQL</li>
            <li>superset_status - 检查服务状态</li>
        </ul>
    </div>
    {{else}}
    <div class="endpoint">
        <h3>Superset MCP服务器 <span class="status unavailable">不可用</span></h3>
        <p>Superset服务器未配置或连接失败</p>
    </div>
    {{end}}

    {{if .PrometheusAvailable}}
    <div class="endpoint">
        <h3>Prometheus MCP服务器 <span class="status available">可用</span></h3>
        <p>提供Prometheus监控数据查询功能</p>
        <p><strong>端点:</strong> <a href="/prometheus/mcp">/prometheus/mcp</a></p>
        <p><strong>可用工具:</strong></p>
        <ul>
            <li>prometheus_query - 执行即时查询</li>
            <li>prometheus_query_range - 执行范围查询</li>
            <li>prometheus_targets - 获取监控目标</li>
            <li>prometheus_status - 检查服务状态</li>
            <li>prometheus_common_metrics - 查询常用指标</li>
            <li>prometheus_list_metrics - 获取所有指标</li>
        </ul>
    </div>
    {{else}}
    <div class="endpoint">
        <h3>Prometheus MCP服务器 <span class="status unavailable">不可用</span></h3>
        <p>Prometheus服务器未配置或连接失败</p>
    </div>
    {{end}}
</body>
</html>`))

// templateData HTML模板数据结构
type templateData struct {
	ServerAddresses     []string
	Port                string
	SupersetAvailable   bool
	PrometheusAvailable bool
}

// Multiplexer HTTP服务器
type Multiplexer struct {
	supersetServer   *mcp.Server
	prometheusServer *mcp.Server
	server           *http.Server
	port             string
	serverAddresses  []string // 缓存的服务器地址
	mu               sync.RWMutex
}

// NewMultiplexer 创建复用服务器
func NewMultiplexer(supersetMCP, prometheusMCP *mcp.Server, port string) *Multiplexer {
	m := &Multiplexer{
		supersetServer:   supersetMCP,
		prometheusServer: prometheusMCP,
		port:             port,
	}

	// 预先获取并缓存服务器地址
	m.serverAddresses = m.getServerAddresses()

	return m
}

// Start 启动服务器
func (m *Multiplexer) Start() error {
	// 使用缓存的服务器地址
	m.mu.RLock()
	serverAddresses := m.serverAddresses
	m.mu.RUnlock()

	// 创建主HTTP服务器
	mux := http.NewServeMux()

	// 注册Superset MCP路由
	if m.supersetServer != nil {
		// 创建HTTP处理器
		supersetHandler := mcp.NewStreamableHTTPHandler(
			func(request *http.Request) *mcp.Server {
				return m.supersetServer
			},
			&mcp.StreamableHTTPOptions{},
		)
		mux.Handle(supersetMCPPath, supersetHandler)
	}

	// 注册Prometheus MCP路由
	if m.prometheusServer != nil {
		// 创建HTTP处理器
		prometheusHandler := mcp.NewStreamableHTTPHandler(
			func(request *http.Request) *mcp.Server {
				return m.prometheusServer
			},
			&mcp.StreamableHTTPOptions{},
		)
		mux.Handle(prometheusMCPPath, prometheusHandler)
	}

	// 添加根路径信息 - 使用预编译模板
	mux.HandleFunc(rootPath, m.handleRoot)

	// 构建地址列表
	addrList := make([]string, 0, len(serverAddresses))
	for _, addr := range serverAddresses {
		addrList = append(addrList, fmt.Sprintf("http://%s:%s", addr, m.port))
	}

	// 显示可用的端点信息（只输出一次）
	if m.supersetServer != nil {
		endpoints := make([]string, 0, len(serverAddresses))
		for _, addr := range serverAddresses {
			endpoints = append(endpoints, fmt.Sprintf("http://%s:%s/superset/mcp", addr, m.port))
		}
		log.Printf("Superset MCP端点: %s", strings.Join(endpoints, ", "))
	}
	if m.prometheusServer != nil {
		endpoints := make([]string, 0, len(serverAddresses))
		for _, addr := range serverAddresses {
			endpoints = append(endpoints, fmt.Sprintf("http://%s:%s/prometheus/mcp", addr, m.port))
		}
		log.Printf("Prometheus MCP端点: %s", strings.Join(endpoints, ", "))
	}

	log.Printf(logServerAddresses, strings.Join(addrList, ", "))

	// 创建优化的HTTP服务器
	m.server = &http.Server{
		Addr:           ":" + m.port,
		Handler:        mux,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		IdleTimeout:    idleTimeout,
		MaxHeaderBytes: maxHeaderBytes,
	}

	return m.server.ListenAndServe()
}

// handleRoot 处理根路径请求
func (m *Multiplexer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != rootPath {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", contentTypeHTML)

	// 使用缓存的服务器地址
	m.mu.RLock()
	serverAddresses := m.serverAddresses
	m.mu.RUnlock()

	// 准备模板数据
	data := templateData{
		ServerAddresses:     serverAddresses,
		Port:                m.port,
		SupersetAvailable:   m.supersetServer != nil,
		PrometheusAvailable: m.prometheusServer != nil,
	}

	// 使用缓冲区来提高性能
	var buf bytes.Buffer
	if err := htmlTemplate.Execute(&buf, data); err != nil {
		log.Printf(logTemplateError, err)
		http.Error(w, httpErrorInternal, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf(logWriteError, err)
	}
}

// Shutdown 优雅关闭服务器
func (m *Multiplexer) Shutdown(ctx context.Context) error {
	if m.server != nil {
		return m.server.Shutdown(ctx)
	}
	return nil
}

// getServerAddresses 获取服务器地址列表
func (m *Multiplexer) getServerAddresses() []string {
	addressSet := make(map[string]bool) // 使用map来去重
	addresses := make([]string, 0, 4)   // 预分配容量

	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf(logInterfaceError, err)
		return []string{"localhost"}
	}

	for _, iface := range interfaces {
		// 跳过回环接口和down状态的接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			log.Printf(logAddressError, iface.Name, err)
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				// 只获取IPv4地址，排除私有网络地址中的某些特殊范围
				if ip4 := ipnet.IP.To4(); ip4 != nil && !ip4.IsLoopback() {
					// 过滤掉Docker等虚拟网络接口的地址
					if !isDockerOrVirtualIP(ip4) {
						ipStr := ip4.String()
						// 使用map去重
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
