package prometheus

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// 常量定义
const (
	defaultConnectionTimeout = 5 * time.Second
	logPrefixQuery           = "Prometheus查询警告 [query=%s]: %v"
	logPrefixRangeQuery      = "Prometheus范围查询警告 [query=%s]: %v"
)

// Client Prometheus客户端
type Client struct {
	client v1.API
}

// NewClient 创建新的Prometheus客户端
func NewClient(serverURL string) (*Client, error) {
	config := api.Config{
		Address:      serverURL,
		RoundTripper: api.DefaultRoundTripper,
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建prometheus客户端失败: %w", err)
	}

	v1api := v1.NewAPI(client)
	return &Client{client: v1api}, nil
}

// QueryInstant 执行即时查询
func (c *Client) QueryInstant(ctx context.Context, query string) (model.Value, error) {
	result, warnings, err := c.client.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}

	if len(warnings) > 0 {
		log.Printf(logPrefixQuery, query, warnings)
	}

	return result, nil
}

// QueryRange 执行范围查询
func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (model.Value, error) {
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}

	result, warnings, err := c.client.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("范围查询失败: %w", err)
	}

	if len(warnings) > 0 {
		log.Printf(logPrefixRangeQuery, query, warnings)
	}

	return result, nil
}

// GetTargets 获取所有目标
func (c *Client) GetTargets(ctx context.Context) (v1.TargetsResult, error) {
	targets, err := c.client.Targets(ctx)
	if err != nil {
		return v1.TargetsResult{}, fmt.Errorf("获取目标失败: %w", err)
	}
	return targets, nil
}

// TestConnection 测试连接
func (c *Client) TestConnection(ctx context.Context) error {
	testCtx, cancel := context.WithTimeout(ctx, defaultConnectionTimeout)
	defer cancel()

	_, _, err := c.client.Query(testCtx, "up", time.Now())
	return err
}

// GetMetricNames 获取指标名称列表
func (c *Client) GetMetricNames(ctx context.Context) ([]string, error) {
	names, _, err := c.client.LabelValues(ctx, "__name__", nil, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		return nil, fmt.Errorf("获取指标名称失败: %w", err)
	}

	result := make([]string, 0, len(names))
	for _, name := range names {
		result = append(result, string(name))
	}

	return result, nil
}

// MetricQueries 预定义的指标查询
var MetricQueries = map[string]string{
	"cpu":     `100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)`,
	"memory":  "(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100",
	"disk":    "(1 - (node_filesystem_avail_bytes{mountpoint=\"/\"} / node_filesystem_size_bytes{mountpoint=\"/\"})) * 100",
	"network": "rate(node_network_receive_bytes_total[5m])",
	"up":      "up",
}
