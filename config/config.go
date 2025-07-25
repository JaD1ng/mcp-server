package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// SupersetConfig 子配置
type SupersetConfig struct {
	URL  string `yaml:"url"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

// PrometheusConfig 子配置
type PrometheusConfig struct {
	URL string `yaml:"url"`
}

// Config 统一配置结构
type Config struct {
	HTTPPort   string           `yaml:"http_port"`
	Timeout    time.Duration    `yaml:"timeout"`
	Superset   SupersetConfig   `yaml:"superset"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
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
	if cfg.HTTPPort == "" {
		cfg.HTTPPort = "8080"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.Superset.URL == "" {
		cfg.Superset.URL = "http://superset.yzh-logverse.k8s.qiniu.io"
	}
	if cfg.Superset.User == "" {
		cfg.Superset.User = "dingnanjia"
	}
	if cfg.Superset.Pass == "" {
		cfg.Superset.Pass = "nanjia123"
	}
	if cfg.Prometheus.URL == "" {
		cfg.Prometheus.URL = "http://hd-piko.prometheus.qiniu.io/"
	}

	return &cfg, nil
}

// LoadConfig 从YAML文件加载配置
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
