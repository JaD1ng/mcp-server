# MCP Server

一个基于Go语言开发的MCP (Model Context Protocol) 服务器，为AI模型提供访问Superset和Prometheus服务的能力。

## 功能特性

- 🔌 **MCP协议支持** - 基于官方Go SDK实现MCP协议
- 📊 **Superset集成** - 提供数据库查询和SQL执行功能
- 📈 **Prometheus集成** - 提供监控数据查询和指标获取功能
- 🛡️ **优雅关闭** - 支持信号处理和优雅关闭

## 项目结构

```
mcp-server/
├── main.go                    # 应用程序入口点
├── config/                    # 配置管理
│   ├── config.go             # 配置结构和加载逻辑
│   └── config.yaml           # 配置文件
├── internal/                  # 内部实现模块
│   ├── common/               # 通用组件
│   │   └── response.go       # 响应处理工具
│   ├── multiplexer/          # HTTP多路复用器
│   │   └── multiplexer.go    # 服务器路由和Web界面
│   ├── superset/             # Superset集成
│   │   ├── superset_client.go    # Superset客户端
│   │   └── superset_server.go    # Superset MCP服务器
│   └── prometheus/           # Prometheus集成
│       ├── prometheus_client.go  # Prometheus客户端
│       └── prometheus_server.go  # Prometheus MCP服务器
├── go.mod                    # Go模块定义
├── go.sum                    # 依赖校验和
└── README.md                 # 项目说明文档
```

## 快速开始

### 环境要求

- Go 1.24.5 或更高版本
- 可访问的Superset实例（可选）
- 可访问的Prometheus实例（可选）

### 安装依赖

```bash
go mod tidy
```

### 配置

编辑 `config/config.yaml` 文件：

```yaml
http_port: "8080"              # HTTP服务端口
timeout: "30s"                 # 请求超时时间
superset:                      # Superset配置
  url: "http://your-superset-url"
  user: "your-username"
  pass: "your-password"
prometheus:                    # Prometheus配置
  url: "http://your-prometheus-url"
```

### 运行服务器

```bash
# 使用默认配置文件
go run main.go

# 或指定配置文件路径
go run main.go -config /path/to/config.yaml
```

### 构建可执行文件

```bash
go build
```

## MCP工具

### Superset工具

| 工具名称 | 描述 | 参数 |
|---------|------|------|
| `superset_list_databases` | 获取所有可用的数据库列表 | 无 |
| `superset_execute_sql` | 在指定数据库中执行SQL查询 | `sql`, `database_id` |
| `superset_execute_sql_with_schema` | 在指定数据库和schema中执行SQL查询 | `sql`, `database_id`, `schema` |
| `superset_status` | 检查Superset服务状态和连接 | 无 |

### Prometheus工具

| 工具名称 | 描述 | 参数 |
|---------|------|------|
| `prometheus_query` | 执行Prometheus即时查询 | `query` |
| `prometheus_query_range` | 执行Prometheus范围查询 | `query`, `start_time`, `end_time`, `step` |
| `prometheus_targets` | 获取Prometheus监控目标 | 无 |
| `prometheus_status` | 检查Prometheus服务状态和连接 | 无 |
| `prometheus_common_metrics` | 查询常用Prometheus指标 | `metric_type` |
| `prometheus_list_metrics` | 获取所有可用的指标名称 | 无 |

## 使用示例

### 访问Web界面

启动服务器后，访问 `http://localhost:8080` 查看服务状态和可用端点。

### MCP端点

- Superset MCP端点: `http://localhost:8080/superset/mcp`
- Prometheus MCP端点: `http://localhost:8080/prometheus/mcp`

### 常用Prometheus指标类型

- `cpu` - CPU使用率
- `memory` - 内存使用率
- `disk` - 磁盘使用率
- `network` - 网络流量
- `up` - 服务可用性

## 故障排除

### 常见问题

1. **连接失败**
   - 检查配置文件中的URL是否正确
   - 确认网络连接正常
   - 验证认证信息

2. **权限错误**
   - 检查Superset用户权限
   - 确认数据库访问权限

3. **超时错误**
   - 调整配置文件中的timeout设置
   - 检查网络延迟
