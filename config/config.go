package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"mcp-server/internal/core"

	"gopkg.in/yaml.v3"
)

// PrometheusConfig Prometheus服务配置
type PrometheusConfig struct {
	Enabled  bool   `yaml:"enabled"`
	URL      string `yaml:"url"`
	Endpoint string `yaml:"endpoint"`
}

// GetType 实现ServiceConfig接口
func (p *PrometheusConfig) GetType() core.ServiceType {
	return core.ServiceTypePrometheus
}

// GetEndpoint 实现ServiceConfig接口
func (p *PrometheusConfig) GetEndpoint() string {
	if p.Endpoint != "" {
		return p.Endpoint
	}
	return "/prometheus/mcp"
}

// IsEnabled 实现ServiceConfig接口
func (p *PrometheusConfig) IsEnabled() bool {
	return p.Enabled && p.URL != ""
}

// Validate 实现ServiceConfig接口
func (p *PrometheusConfig) Validate() error {
	if p.Enabled && p.URL == "" {
		return fmt.Errorf("prometheus服务已启用但URL为空")
	}
	return nil
}

// SupersetConfig Superset服务配置
type SupersetConfig struct {
	Enabled  bool   `yaml:"enabled"`
	URL      string `yaml:"url"`
	User     string `yaml:"user"`
	Pass     string `yaml:"pass"`
	Endpoint string `yaml:"endpoint"`
}

// GetType 实现ServiceConfig接口
func (s *SupersetConfig) GetType() core.ServiceType {
	return core.ServiceTypeSuperset
}

// GetEndpoint 实现ServiceConfig接口
func (s *SupersetConfig) GetEndpoint() string {
	if s.Endpoint != "" {
		return s.Endpoint
	}
	return "/superset/mcp"
}

// IsEnabled 实现ServiceConfig接口
func (s *SupersetConfig) IsEnabled() bool {
	return s.Enabled && s.URL != ""
}

// Validate 实现ServiceConfig接口
func (s *SupersetConfig) Validate() error {
	if s.Enabled {
		if s.URL == "" {
			return fmt.Errorf("superset服务已启用但URL为空")
		}
		if s.User == "" {
			return fmt.Errorf("superset服务已启用但用户名为空")
		}
		if s.Pass == "" {
			return fmt.Errorf("superset服务已启用但密码为空")
		}
	}
	return nil
}

// Config 应用程序配置
type Config struct {
	HTTPPort   string            `yaml:"http_port"`
	Timeout    time.Duration     `yaml:"timeout"`
	Prometheus *PrometheusConfig `yaml:"prometheus"`
	Superset   *SupersetConfig   `yaml:"superset"`
}

// GetServices 获取启用的服务配置列表 (保持向后兼容)
func (c *Config) GetServices() []core.ServiceConfig {
	// 使用新的函数式API
	return FilterEnabledServices(c)
}

// Validate 验证配置 (保持向后兼容)
func (c *Config) Validate() error {
	// 使用新的函数式验证API
	result := ValidateConfig(c)
	if !result.IsValid() {
		return result.GetFirstError()
	}
	return nil
}

// LoadConfigFromYAML 从YAML文件加载配置
func LoadConfigFromYAML(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("无法打开配置文件: %w", err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("YAML解析失败: %w", err)
	}

	// 设置默认值
	setDefaults(&cfg)

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &cfg, nil
}

// setDefaults 设置默认配置值
func setDefaults(cfg *Config) {
	if cfg.HTTPPort == "" {
		cfg.HTTPPort = "8080"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// 初始化Prometheus配置
	if cfg.Prometheus == nil {
		cfg.Prometheus = &PrometheusConfig{}
	}
	if cfg.Prometheus.URL == "" {
		cfg.Prometheus.URL = "http://hd-piko.prometheus.qiniu.io/"
		cfg.Prometheus.Enabled = true
	}

	// 初始化Superset配置
	if cfg.Superset == nil {
		cfg.Superset = &SupersetConfig{}
	}
	if cfg.Superset.URL == "" {
		cfg.Superset.URL = "http://superset.yzh-logverse.k8s.qiniu.io"
		cfg.Superset.User = "dingnanjia"
		cfg.Superset.Pass = "nanjia123"
		cfg.Superset.Enabled = true
	}
}

// LoadConfig 加载配置
func LoadConfig() *Config {
	configPath := flag.String("config", "config/config.yaml", "YAML配置文件路径")
	flag.Parse()

	cfg, err := LoadConfigFromYAML(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: 无法加载配置文件 %s: %v\n", *configPath, err)
		fmt.Fprintf(os.Stderr, "请确保配置文件存在且格式正确\n")
		os.Exit(1)
	}

	return cfg
}
