package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"mcp-server/config"
	"mcp-server/internal/multiplexer"
	"mcp-server/internal/prometheus"
	"mcp-server/internal/superset"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// serverInitResult 服务器初始化结果
type serverInitResult struct {
	supersetServer   *superset.SupersetMCPServer
	prometheusServer *prometheus.PrometheusMCPServer
}

// main 主函数 - 应用程序入口点
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 加载配置
	cfg := config.LoadConfig()

	// 打印启动信息
	printStartupInfo(cfg)

	// 创建上下文用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 并发初始化服务器
	result := initializeServers(ctx, cfg)

	// 验证至少有一个服务器可用
	if result.supersetServer == nil && result.prometheusServer == nil {
		log.Fatal("错误: 没有可用的MCP服务器，无法启动服务")
	}

	// 创建并启动多路复用服务器
	multiplexerServer := createMultiplexer(result, cfg.HTTPPort)

	// 启动服务器并等待关闭信号
	runServer(multiplexerServer)
}

// printStartupInfo 打印启动信息
func printStartupInfo(cfg *config.Config) {
	log.Printf("启动MCP服务器...")
	log.Printf("配置信息:")
	log.Printf("- HTTP端口: %s", cfg.HTTPPort)
	log.Printf("- 超时时间: %v", cfg.Timeout)
	log.Printf("- Superset URL: %s", cfg.Superset.URL)
	log.Printf("- Prometheus URL: %s", cfg.Prometheus.URL)
}

// initializeServers 并发初始化所有服务器
func initializeServers(ctx context.Context, cfg *config.Config) *serverInitResult {
	var wg sync.WaitGroup
	result := &serverInitResult{}

	// 并发初始化Superset服务器
	if cfg.Superset.URL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result.supersetServer = initializeSupersetServer(ctx, cfg)
		}()
	} else {
		log.Printf("跳过Superset服务器创建 (URL未配置)")
	}

	// 并发初始化Prometheus服务器
	if cfg.Prometheus.URL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result.prometheusServer = initializePrometheusServer(ctx, cfg)
		}()
	} else {
		log.Printf("跳过Prometheus服务器创建 (URL未配置)")
	}

	// 等待所有初始化完成
	wg.Wait()
	return result
}

// initializeSupersetServer 初始化Superset服务器
func initializeSupersetServer(ctx context.Context, cfg *config.Config) *superset.SupersetMCPServer {
	// 创建客户端
	client, err := superset.NewClient(cfg.Superset.URL, cfg.Superset.User, cfg.Superset.Pass, cfg.Timeout)
	if err != nil {
		log.Printf("警告: 创建Superset客户端失败: %v", err)
		return nil
	}

	// 测试连接
	if err := testSupersetConnection(ctx, client); err != nil {
		log.Printf("警告: Superset连接测试失败: %v", err)
	} else {
		log.Printf("✓ Superset连接正常")
	}

	// 创建MCP服务器
	server := superset.NewSupersetMCPServer(client)
	log.Printf("✓ Superset MCP服务器已创建")
	return server
}

// initializePrometheusServer 初始化Prometheus服务器
func initializePrometheusServer(ctx context.Context, cfg *config.Config) *prometheus.PrometheusMCPServer {
	// 创建客户端
	client, err := prometheus.NewClient(cfg.Prometheus.URL)
	if err != nil {
		log.Printf("警告: 创建Prometheus客户端失败: %v", err)
		return nil
	}

	// 测试连接
	if err := testPrometheusConnection(ctx, client); err != nil {
		log.Printf("警告: Prometheus连接测试失败: %v", err)
	} else {
		log.Printf("✓ Prometheus连接正常")
	}

	// 创建MCP服务器
	server := prometheus.NewPrometheusMCPServer(client)
	log.Printf("✓ Prometheus MCP服务器已创建")
	return server
}

// testSupersetConnection 测试Superset连接
func testSupersetConnection(ctx context.Context, client *superset.Client) error {
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return client.TestConnection(testCtx)
}

// testPrometheusConnection 测试Prometheus连接
func testPrometheusConnection(ctx context.Context, client *prometheus.Client) error {
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return client.TestConnection(testCtx)
}

// createMultiplexer 创建多路复用服务器
func createMultiplexer(result *serverInitResult, port string) *multiplexer.Multiplexer {
	var supersetMCP, prometheusMCP *mcp.Server

	if result.supersetServer != nil {
		supersetMCP = result.supersetServer.GetServer()
	}

	if result.prometheusServer != nil {
		prometheusMCP = result.prometheusServer.GetServer()
	}

	multiplexerServer := multiplexer.NewMultiplexer(supersetMCP, prometheusMCP, port)

	return multiplexerServer
}

// runServer 运行服务器并处理关闭信号
func runServer(multiplexerServer *multiplexer.Multiplexer) {
	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器
	go func() {
		if err := multiplexerServer.Start(); err != nil {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// 等待关闭信号
	<-sigChan
	log.Printf("收到关闭信号，正在关闭...")

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := multiplexerServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("关闭服务器时出错: %v", err)
	} else {
		log.Printf("服务器已关闭")
	}
}
