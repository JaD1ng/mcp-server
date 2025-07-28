package core

import (
	"strings"
)

// UnsupportedServiceError 不支持的服务类型错误
type UnsupportedServiceError struct {
	ServiceType ServiceType
	message     string // 缓存错误信息
}

func (e *UnsupportedServiceError) Error() string {
	if e.message == "" {
		e.message = "不支持的服务类型: " + string(e.ServiceType)
	}
	return e.message
}

// NewUnsupportedServiceError 创建不支持的服务类型错误
func NewUnsupportedServiceError(serviceType ServiceType) *UnsupportedServiceError {
	return &UnsupportedServiceError{ServiceType: serviceType}
}

// ServiceCreationError 服务创建错误
type ServiceCreationError struct {
	ServiceType ServiceType
	Err         error
	message     string // 缓存错误信息
}

func (e *ServiceCreationError) Error() string {
	if e.message == "" {
		// 预构建错误信息以提高性能
		var builder strings.Builder
		builder.WriteString("创建服务失败 [")
		builder.WriteString(string(e.ServiceType))
		builder.WriteString("]: ")
		if e.Err != nil {
			builder.WriteString(e.Err.Error())
		}
		e.message = builder.String()
	}
	return e.message
}

func (e *ServiceCreationError) Unwrap() error {
	return e.Err
}

// NewServiceCreationError 创建服务创建错误
func NewServiceCreationError(serviceType ServiceType, err error) *ServiceCreationError {
	return &ServiceCreationError{
		ServiceType: serviceType,
		Err:         err,
	}
}

// ConnectionError 连接错误
type ConnectionError struct {
	ServiceType ServiceType
	Endpoint    string
	Err         error
	message     string // 缓存错误信息
}

func (e *ConnectionError) Error() string {
	if e.message == "" {
		// 使用strings.Builder提高字符串拼接性能
		var builder strings.Builder
		builder.WriteString("连接失败 [")
		builder.WriteString(string(e.ServiceType))
		builder.WriteString("] ")
		builder.WriteString(e.Endpoint)
		builder.WriteString(": ")
		if e.Err != nil {
			builder.WriteString(e.Err.Error())
		}
		e.message = builder.String()
	}
	return e.message
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// NewConnectionError 创建连接错误
func NewConnectionError(serviceType ServiceType, endpoint string, err error) *ConnectionError {
	return &ConnectionError{
		ServiceType: serviceType,
		Endpoint:    endpoint,
		Err:         err,
	}
}
