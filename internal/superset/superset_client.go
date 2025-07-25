package superset

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// 常量定义
const (
	// API端点
	loginEndpoint      = "/login/"
	apiEndpoint        = "/api/v1"
	healthEndpoint     = "/health"
	databaseEndpoint   = "/api/v1/database/"
	sqlExecuteEndpoint = "/api/v1/sqllab/execute/"

	// HTTP头常量
	contentTypeJSON = "application/json"
	contentTypeForm = "application/x-www-form-urlencoded"
	headerAccept    = "Accept"
	headerCSRF      = "X-CSRFToken"
	headerReferer   = "Referer"

	// CSRF令牌缓存时间
	csrfTokenCacheDuration = 5 * time.Minute
	
	// HTTP传输层配置
	maxIdleConns        = 100
	maxIdleConnsPerHost = 10
	maxConnsPerHost     = 50
	idleConnTimeout     = 90 * time.Second
	tlsHandshakeTimeout = 10 * time.Second
	responseHeaderTimeout = 30 * time.Second
)

// CSRF令牌正则表达式 - 预编译提升性能
var csrfTokenRegex = regexp.MustCompile(`name="csrf_token"[^>]*value="([^"]*)"`)

// Database 数据库结构
type Database struct {
	ID            int    `json:"id"`
	DatabaseName  string `json:"database_name"`
	Backend       string `json:"backend"`
	SQLAlchemyURI string `json:"sqlalchemy_uri"`
	CreatedOn     string `json:"created_on"`
	ChangedOn     string `json:"changed_on"`
}

// SQLResult SQL执行结果
type SQLResult struct {
	Columns []string `json:"columns"`
	Data    [][]any  `json:"data"`
	Query   string   `json:"query"`
	Status  string   `json:"status"`
}

// csrfTokenCache CSRF令牌缓存
type csrfTokenCache struct {
	token     string
	expiresAt time.Time
}

// Client Superset客户端
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	loggedIn   bool
	mu         sync.RWMutex
	timeout    time.Duration
	csrfCache  csrfTokenCache
	sqlLabURL  string // 缓存的sqllab URL
}

// NewClient 创建新的Superset客户端
func NewClient(baseURL, username, password string, timeout time.Duration) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("创建cookie jar失败: %w", err)
	}

	// 创建优化的HTTP传输层
	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,              // 最大空闲连接数
		MaxIdleConnsPerHost: maxIdleConnsPerHost,       // 每个主机的最大空闲连接数
		IdleConnTimeout:     idleConnTimeout,           // 空闲连接超时
		TLSHandshakeTimeout: tlsHandshakeTimeout,       // TLS握手超时
		DisableCompression:  false,                     // 启用压缩
		ForceAttemptHTTP2:   true,                      // 强制尝试HTTP/2
		// 添加更多优化配置
		MaxConnsPerHost:       maxConnsPerHost,         // 每个主机的最大连接数
		ResponseHeaderTimeout: responseHeaderTimeout,   // 响应头超时
	}

	return &Client{
		baseURL:   baseURL,
		username:  username,
		password:  password,
		sqlLabURL: baseURL + "/superset/sqllab", // 缓存常用URL
		httpClient: &http.Client{
			Timeout:   timeout,
			Jar:       jar,
			Transport: transport,
		},
		timeout: timeout,
	}, nil
}

// TestConnection 测试连接
func (c *Client) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+healthEndpoint, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器响应异常，状态码: %d", resp.StatusCode)
	}

	return nil
}

// getCSRFToken 获取CSRF令牌（带缓存）
func (c *Client) getCSRFToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	// 检查缓存是否有效
	if c.csrfCache.token != "" && time.Now().Before(c.csrfCache.expiresAt) {
		token := c.csrfCache.token
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	// 缓存失效，重新获取
	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查，防止并发重复请求
	if c.csrfCache.token != "" && time.Now().Before(c.csrfCache.expiresAt) {
		return c.csrfCache.token, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+loginEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取登录页面失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	matches := csrfTokenRegex.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("未找到CSRF令牌")
	}

	// 缓存令牌
	token := matches[1]
	c.csrfCache = csrfTokenCache{
		token:     token,
		expiresAt: time.Now().Add(csrfTokenCacheDuration),
	}

	return token, nil
}

// Login 登录
func (c *Client) Login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.loggedIn {
		return nil
	}

	csrfToken, err := c.getCSRFTokenForLogin(ctx)
	if err != nil {
		return fmt.Errorf("获取CSRF令牌失败: %w", err)
	}

	formData := url.Values{
		"username":   {c.username},
		"password":   {c.password},
		"csrf_token": {csrfToken},
	}

	formBytes := []byte(formData.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+loginEndpoint, bytes.NewReader(formBytes))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", contentTypeForm)
	req.Header.Set(headerReferer, c.baseURL+loginEndpoint)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查登录成功
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusSeeOther {
		location := resp.Header.Get("Location")
		if c.isSuccessfulRedirect(location) {
			c.loggedIn = true
			return nil
		}
	}

	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		if c.isLoginError(bodyStr) {
			return fmt.Errorf("用户名或密码错误")
		}

		if c.isLoginSuccess(bodyStr) {
			c.loggedIn = true
			return nil
		}

		return fmt.Errorf("登录失败")
	}

	return fmt.Errorf("登录失败，状态码: %d", resp.StatusCode)
}

// getCSRFTokenForLogin 为登录专门获取CSRF令牌（不使用缓存）
func (c *Client) getCSRFTokenForLogin(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+loginEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取登录页面失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	matches := csrfTokenRegex.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("未找到CSRF令牌")
	}

	return matches[1], nil
}

// isSuccessfulRedirect 检查是否为成功的重定向
func (c *Client) isSuccessfulRedirect(location string) bool {
	return strings.Contains(location, "/superset/welcome") ||
		location == "/" ||
		strings.Contains(location, "/superset")
}

// isLoginError 检查是否为登录错误
func (c *Client) isLoginError(body string) bool {
	return strings.Contains(body, "Invalid login") ||
		strings.Contains(body, "Invalid username or password") ||
		strings.Contains(body, "Authentication failed")
}

// isLoginSuccess 检查是否登录成功
func (c *Client) isLoginSuccess(body string) bool {
	return strings.Contains(body, "superset") && strings.Contains(body, "dashboard")
}

// ensureLoggedIn 确保已登录
func (c *Client) ensureLoggedIn(ctx context.Context) error {
	c.mu.RLock()
	if c.loggedIn {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	return c.Login(ctx)
}

// GetDatabases 获取数据库列表
func (c *Client) GetDatabases(ctx context.Context) ([]Database, error) {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return nil, fmt.Errorf("登录失败: %w", err)
	}

	csrfToken, err := c.getCSRFToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取CSRF令牌失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+databaseEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set(headerAccept, contentTypeJSON)
	req.Header.Set(headerCSRF, csrfToken)
	req.Header.Set(headerReferer, c.sqlLabURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取数据库列表失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result []Database `json:"result"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		// 尝试直接解析为数组
		var databases []Database
		if err := json.Unmarshal(body, &databases); err != nil {
			return nil, fmt.Errorf("解析响应失败: %w, 响应体: %s", err, string(body))
		}
		return databases, nil
	}

	return result.Result, nil
}

// ExecuteSQL 执行SQL查询
func (c *Client) ExecuteSQL(ctx context.Context, sql string, databaseID int) (*SQLResult, error) {
	return c.executeSQLInternal(ctx, sql, databaseID, "")
}

// ExecuteSQLWithSchema 执行带schema的SQL查询
func (c *Client) ExecuteSQLWithSchema(ctx context.Context, sql string, databaseID int, schema string) (*SQLResult, error) {
	return c.executeSQLInternal(ctx, sql, databaseID, schema)
}

// executeSQLInternal 内部SQL执行方法
func (c *Client) executeSQLInternal(ctx context.Context, sql string, databaseID int, schema string) (*SQLResult, error) {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return nil, fmt.Errorf("登录失败: %w", err)
	}

	csrfToken, err := c.getCSRFToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取CSRF令牌失败: %w", err)
	}

	payload := map[string]any{
		"database_id": databaseID,
		"sql":         sql,
		"schema":      schema,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+sqlExecuteEndpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set(headerAccept, contentTypeJSON)
	req.Header.Set(headerCSRF, csrfToken)
	req.Header.Set(headerReferer, c.sqlLabURL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("执行SQL失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var supersetResponse struct {
		QueryID int              `json:"query_id"`
		Status  string           `json:"status"`
		Data    []map[string]any `json:"data"`
		Columns []struct {
			ColumnName string `json:"column_name"`
			Name       string `json:"name"`
			Type       string `json:"type"`
		} `json:"columns"`
		Query struct {
			SQL string `json:"sql"`
		} `json:"query"`
	}

	if err := json.Unmarshal(body, &supersetResponse); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应体: %s", err, string(body))
	}

	// 预分配切片容量以提升性能
	columns := make([]string, 0, len(supersetResponse.Columns))
	for _, col := range supersetResponse.Columns {
		columns = append(columns, col.Name)
	}

	data := make([][]any, 0, len(supersetResponse.Data))
	for _, row := range supersetResponse.Data {
		rowData := make([]any, 0, len(supersetResponse.Columns))
		for _, col := range supersetResponse.Columns {
			rowData = append(rowData, row[col.Name])
		}
		data = append(data, rowData)
	}

	return &SQLResult{
		Columns: columns,
		Data:    data,
		Query:   supersetResponse.Query.SQL,
		Status:  supersetResponse.Status,
	}, nil
}
