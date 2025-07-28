# MCP服务器

一个基于[MCP (Model Context Protocol)](https://modelcontextprotocol.io/)协议的多服务集成服务器，使用Go语言开发。该服务器提供了Prometheus监控和Superset数据查询的统一接口。

## 项目简介

MCP服务器是一个轻量级的HTTP服务器，通过MCP协议为AI助手提供访问Prometheus监控数据和Superset数据查询的能力。支持多服务并发初始化、配置文件驱动和优雅关闭等特性。

## 功能特性

### 核心功能
- 🚀 **多服务支持**: 同时支持Prometheus和Superset服务
- ⚡ **并发初始化**: 服务并发启动，提高启动速度
- 🔧 **配置驱动**: 通过YAML配置文件管理服务
- 🛡️ **优雅关闭**: 支持安全的服务停止和资源清理
- 🌐 **HTTP接口**: 提供RESTful API访问
- 🔍 **连接测试**: 自动检测服务连接状态

### Prometheus服务功能
- 📊 **即时查询**: 执行PromQL即时查询
- 📈 **范围查询**: 执行时间范围查询
- 🎯 **监控目标**: 获取监控目标列表
- ✅ **状态检查**: 检查Prometheus服务状态
- 📋 **常用指标**: 查询CPU、内存、磁盘等常用指标
- 📝 **指标列表**: 获取所有可用指标名称

### Superset服务功能
- 🗃️ **数据库列表**: 获取所有可用数据库
- 💻 **SQL执行**: 在指定数据库中执行SQL查询
- 🏗️ **Schema支持**: 支持指定数据库和schema执行查询
- ✅ **状态检查**: 检查Superset服务状态

## 技术栈

- **语言**: Go 1.24.5+
- **协议**: MCP (Model Context Protocol)
- **配置**: YAML
- **依赖管理**: Go Modules
- **外部集成**: Prometheus API, Superset API

## 快速开始

### 环境要求

- Go 1.24.5 或更高版本
- 可访问的Prometheus实例
- 可访问的Superset实例

### 安装与构建

1. **克隆项目**:
   ```bash
   git clone <repository-url>
   cd mcp-server
   ```

2. **安装依赖**:
   ```bash
   make tidy
   ```

3. **构建项目**:
   ```bash
   make build
   ```

### 配置

1. **创建配置文件**: 复制并修改配置文件
   ```bash
   cp config/config.yaml config/config.yaml.local
   ```

2. **编辑配置**: 修改 `config/config.yaml` 文件
   ```yaml
   # HTTP服务器配置
   http_port: "8080"
   timeout: 30s

   # Prometheus监控服务配置
   prometheus:
     enabled: true
     url: "http://your-prometheus-server:9090/"
     endpoint: "/prometheus/mcp"

   # Superset数据查询服务配置
   superset:
     enabled: true
     url: "http://your-superset-server"
     user: "your-username"
     pass: "your-password" 
     endpoint: "/superset/mcp"
   ```

### 运行

1. **直接运行**:
   ```bash
   make run
   ```

2. **开发模式运行**:
   ```bash
   make run-direct
   ```

3. **使用自定义配置**:
   ```bash
   ./bin/mcp-server -config=/path/to/your/config.yaml
   ```

## 使用方法

### API端点

服务器启动后，各服务将在以下端点提供服务：

- **Prometheus服务**: `http://localhost:8080/prometheus/mcp`
- **Superset服务**: `http://localhost:8080/superset/mcp`

### 可用工具

#### Prometheus工具

| 工具名称 | 描述 | 参数 |
|---------|------|------|
| `prometheus_query` | 执行即时查询 | `query`: PromQL查询语句 |
| `prometheus_query_range` | 执行范围查询 | `query`, `start_time`, `end_time`, `step` |
| `prometheus_targets` | 获取监控目标 | 无参数 |
| `prometheus_status` | 检查服务状态 | 无参数 |
| `prometheus_common_metrics` | 查询常用指标 | `metric_type`: cpu/memory/disk/network/up |
| `prometheus_list_metrics` | 获取指标列表 | 无参数 |

#### Superset工具

| 工具名称 | 描述 | 参数 |
|---------|------|------|
| `superset_list_databases` | 获取数据库列表 | 无参数 |
| `superset_execute_sql` | 执行SQL查询 | `sql`, `database_id` |
| `superset_execute_sql_with_schema` | 执行SQL查询(带schema) | `sql`, `database_id`, `schema` |
| `superset_status` | 检查服务状态 | 无参数 |

### 示例

#### 查询Prometheus指标
```bash
curl -X POST "http://localhost:8080/prometheus/mcp" \
  -H "Content-Type: application/json" \
  -d '{"tool": "prometheus_query", "arguments": {"query": "up"}}'
```

#### 执行Superset SQL查询
```bash
curl -X POST "http://localhost:8080/superset/mcp" \
  -H "Content-Type: application/json" \
  -d '{"tool": "superset_execute_sql", "arguments": {"sql": "SELECT * FROM table LIMIT 10", "database_id": "1"}}'
```

## 开发

### 开发命令

```bash
# 格式化代码
make fmt

# 代码检查
make vet

# 完整的开发流程
make dev

# 构建开发版本（包含调试信息）
make build-dev

# 跨平台构建
make build-cross

# 查看所有可用命令
make help
```

### 项目结构

```
mcp-server/
├── cmd/mcp-server/          # 主程序入口
├── config/                  # 配置文件和配置逻辑
├── internal/                # 内部包
│   ├── common/             # 通用响应处理
│   ├── core/               # 核心类型和错误处理
│   ├── multiplexer/        # HTTP服务器和多路复用
│   └── services/           # 服务实现
│       ├── prometheus/     # Prometheus服务
│       ├── superset/       # Superset服务
│       └── registry.go     # 服务注册
├── bin/                    # 构建输出目录
├── Makefile               # 构建脚本
└── README.md              # 项目文档
```

### 添加新服务

1. 在 `internal/services/` 目录下创建新的服务包
2. 实现 `core.Service` 接口
3. 在 `internal/services/registry.go` 中注册服务工厂
4. 在 `config/config.go` 中添加配置结构

## 配置参考

### 完整配置示例

```yaml
# HTTP服务器配置
http_port: "8080"        # HTTP监听端口
timeout: 30s             # 请求超时时间

# Prometheus监控服务
prometheus:
  enabled: true                                    # 是否启用服务
  url: "http://your-prometheus-server:9090/"      # Prometheus服务器URL
  endpoint: "/prometheus/mcp"                     # HTTP端点路径（可选）

# Superset数据查询服务  
superset:
  enabled: true                                    # 是否启用服务
  url: "http://your-superset-server"              # Superset服务器URL
  user: "your-username"                           # 登录用户名
  pass: "your-password"                           # 登录密码
  endpoint: "/superset/mcp"                       # HTTP端点路径（可选）
```

### 配置说明

- `enabled: false` 可以禁用对应服务
- `url` 为空时该服务将被跳过
- `endpoint` 可以自定义服务的HTTP端点路径
- 服务配置支持环境变量替换

## 部署

### 二进制部署

1. 构建可执行文件:
   ```bash
   make build
   ```

2. 复制文件到目标服务器:
   ```bash
   scp bin/mcp-server user@server:/usr/local/bin/
   scp config/config.yaml user@server:/etc/mcp-server/
   ```

3. 运行服务:
   ```bash
   /usr/local/bin/mcp-server -config=/etc/mcp-server/config.yaml
   ```

### Docker部署

创建 `Dockerfile`:
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN make build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bin/mcp-server .
COPY --from=builder /app/config/config.yaml ./config/
EXPOSE 8080
CMD ["./mcp-server"]
```

## 故障排除

### 常见问题

1. **服务连接失败**:
   - 检查配置文件中的URL是否正确
   - 确认网络连接和防火墙设置
   - 验证认证信息是否正确

2. **启动失败**:
   - 检查端口是否被占用
   - 验证配置文件格式
   - 查看详细错误日志

3. **性能问题**:
   - 调整timeout配置
   - 检查目标服务的性能
   - 考虑增加并发限制

### 日志级别

程序使用标准的Go log包，所有重要操作都会记录日志：
- 启动信息和配置
- 服务初始化状态
- 连接测试结果
- 错误和警告信息

## 贡献

欢迎提交Issue和Pull Request！

## 许可证

[在此添加您的许可证信息]

## 更新日志

### v1.0.0
- 初始版本发布
- 支持Prometheus和Superset服务
- 基础的MCP协议实现
- 配置文件支持 