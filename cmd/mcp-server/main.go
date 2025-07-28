package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"mcp-server/config"
	"mcp-server/internal/core"
	"mcp-server/internal/multiplexer"
	_ "mcp-server/internal/services" // 导入以确保init()函数执行，注册服务工厂
)

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

	// 创建多路复用服务器
	server := multiplexer.NewServer(cfg.HTTPPort)

	// 并发初始化和注册服务
	if err := initializeAndRegisterServices(ctx, cfg, server); err != nil {
		log.Fatalf("初始化服务失败: %v", err)
	}

	// 启动服务器并等待关闭信号
	runServer(server)
}

// printStartupInfo 打印启动信息
func printStartupInfo(cfg *config.Config) {
	log.Printf("启动MCP服务器...")
	log.Printf("配置信息:")
	log.Printf("- HTTP端口: %s", cfg.HTTPPort)
	log.Printf("- 超时时间: %v", cfg.Timeout)

	// 打印启用的服务
	services := cfg.GetServices()
	log.Printf("- 启用的服务数量: %d", len(services))
	for _, service := range services {
		log.Printf("  * %s: %s", service.GetType(), service.GetEndpoint())
	}
}

// initializeAndRegisterServices 并发初始化并注册所有服务
func initializeAndRegisterServices(ctx context.Context, cfg *config.Config, server *multiplexer.Server) error {
	// 使用新的函数式API获取服务配置
	serviceConfigs := config.FilterEnabledServices(cfg)

	if len(serviceConfigs) == 0 {
		return fmt.Errorf("没有启用的服务配置")
	}

	var wg sync.WaitGroup
	serviceChan := make(chan core.Service, len(serviceConfigs))
	errorChan := make(chan error, len(serviceConfigs))

	// 并发创建服务
	for _, serviceConfig := range serviceConfigs {
		wg.Add(1)
		go func(config core.ServiceConfig) {
			defer wg.Done()

			log.Printf("初始化服务: %s", config.GetType())

			// 使用新的函数式API创建服务实例
			service, err := core.CreateService(config, cfg.Timeout)
			if err != nil {
				errorChan <- fmt.Errorf("创建服务 %s 失败: %w", config.GetType(), err)
				return
			}

			// 测试连接
			if err := testServiceConnection(ctx, service); err != nil {
				log.Printf("警告: %s 连接测试失败: %v", service.GetType(), err)
			} else {
				log.Printf("✓ %s 连接正常", service.GetType())
			}

			serviceChan <- service
		}(serviceConfig)
	}

	// 等待所有服务初始化完成
	go func() {
		wg.Wait()
		close(serviceChan)
		close(errorChan)
	}()

	// 收集结果
	var services []core.Service
	var errors []error

	for service := range serviceChan {
		services = append(services, service)
	}

	for err := range errorChan {
		errors = append(errors, err)
	}

	// 注册成功创建的服务
	for _, service := range services {
		server.AddService(service)
	}

	// 如果有错误但至少有一个服务成功，记录警告
	if len(errors) > 0 {
		for _, err := range errors {
			log.Printf("警告: %v", err)
		}
	}

	// 如果没有任何服务成功创建，返回错误
	if len(services) == 0 {
		return fmt.Errorf("没有成功创建任何服务")
	}

	log.Printf("✓ 成功初始化 %d 个服务", len(services))
	return nil
}

// testServiceConnection 测试服务连接
func testServiceConnection(ctx context.Context, service core.Service) error {
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return service.TestConnection(testCtx)
}

// runServer 运行服务器并处理关闭信号
func runServer(server *multiplexer.Server) {
	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// 等待关闭信号
	<-sigChan
	log.Printf("收到关闭信号，正在关闭...")

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("关闭服务器时出错: %v", err)
	} else {
		log.Printf("服务器已关闭")
	}
}
