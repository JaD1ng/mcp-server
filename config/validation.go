package config

import (
	"fmt"

	"mcp-server/internal/core"
)

// ValidationError 配置验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("配置验证失败 [%s]: %s", e.Field, e.Message)
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// IsValid 检查验证结果是否有效
func (vr ValidationResult) IsValid() bool {
	return vr.Valid && len(vr.Errors) == 0
}

// GetFirstError 获取第一个错误
func (vr ValidationResult) GetFirstError() error {
	if len(vr.Errors) > 0 {
		return vr.Errors[0]
	}
	return nil
}

// 纯函数验证器

// ValidatePrometheusConfig 验证Prometheus配置 (纯函数)
func ValidatePrometheusConfig(config *PrometheusConfig) ValidationResult {
	var errors []ValidationError

	if config == nil {
		return ValidationResult{Valid: false, Errors: []ValidationError{
			{Field: "prometheus", Message: "配置不能为空"},
		}}
	}

	if config.Enabled && config.URL == "" {
		errors = append(errors, ValidationError{
			Field:   "prometheus.url",
			Message: "服务已启用但URL为空",
		})
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ValidateSupersetConfig 验证Superset配置 (纯函数)
func ValidateSupersetConfig(config *SupersetConfig) ValidationResult {
	var errors []ValidationError

	if config == nil {
		return ValidationResult{Valid: false, Errors: []ValidationError{
			{Field: "superset", Message: "配置不能为空"},
		}}
	}

	if config.Enabled {
		if config.URL == "" {
			errors = append(errors, ValidationError{
				Field:   "superset.url",
				Message: "服务已启用但URL为空",
			})
		}
		if config.User == "" {
			errors = append(errors, ValidationError{
				Field:   "superset.user",
				Message: "服务已启用但用户名为空",
			})
		}
		if config.Pass == "" {
			errors = append(errors, ValidationError{
				Field:   "superset.pass",
				Message: "服务已启用但密码为空",
			})
		}
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// ValidateConfig 验证完整配置 (纯函数)
func ValidateConfig(config *Config) ValidationResult {
	var allErrors []ValidationError

	if config == nil {
		return ValidationResult{Valid: false, Errors: []ValidationError{
			{Field: "config", Message: "配置不能为空"},
		}}
	}

	// 验证Prometheus配置
	if promResult := ValidatePrometheusConfig(config.Prometheus); !promResult.IsValid() {
		allErrors = append(allErrors, promResult.Errors...)
	}

	// 验证Superset配置
	if supersetResult := ValidateSupersetConfig(config.Superset); !supersetResult.IsValid() {
		allErrors = append(allErrors, supersetResult.Errors...)
	}

	return ValidationResult{
		Valid:  len(allErrors) == 0,
		Errors: allErrors,
	}
}

// FilterEnabledServices 过滤启用的服务配置 (纯函数)
func FilterEnabledServices(config *Config) []core.ServiceConfig {
	if config == nil {
		return []core.ServiceConfig{}
	}

	var services []core.ServiceConfig

	if config.Prometheus != nil && config.Prometheus.IsEnabled() {
		services = append(services, config.Prometheus)
	}

	if config.Superset != nil && config.Superset.IsEnabled() {
		services = append(services, config.Superset)
	}

	return services
}

// ValidateServiceConfig 验证单个服务配置 (纯函数)
func ValidateServiceConfig(serviceConfig core.ServiceConfig) ValidationResult {
	switch config := serviceConfig.(type) {
	case *PrometheusConfig:
		return ValidatePrometheusConfig(config)
	case *SupersetConfig:
		return ValidateSupersetConfig(config)
	default:
		return ValidationResult{Valid: false, Errors: []ValidationError{
			{Field: "service", Message: fmt.Sprintf("未知的服务配置类型: %T", serviceConfig)},
		}}
	}
}
