# MCP论文检索服务 - 代理指南

本文档为编程代理提供详细的项目信息，包括构建步骤、代码结构、测试约定和扩展指南。
api实现参考自：https://github.com/binary-husky/gpt_academic/blob/master/crazy_functions/review_fns/data_sources
## 项目概述

这是一个基于Go语言实现的MCP (Model Context Protocol) 论文检索服务，目前支持arXiv论文搜索，计划扩展支持其他学术论文数据库。

### 核心功能
- 🔍 arXiv论文搜索和详情获取
- 🌐 MCP协议完整支持
- 📊 结构化JSON数据输出
- 🚀 高性能并发处理
- 📝 详细的日志记录

### 技术栈
- **Go**: 1.24+
- **MCP SDK**: github.com/modelcontextprotocol/go-sdk v0.3.1
- **HTTP客户端**: net/http (标准库)
- **XML解析**: encoding/xml (标准库)

## 项目结构

```
mcp/
├── main.go              # MCP服务器主程序和工具注册
├── logging.go           # HTTP请求日志中间件
├── go.mod               # Go模块定义和依赖
├── go.sum               # 依赖校验文件
├── arxiv/               # arXiv相关实现
│   ├── client.go        # arXiv API客户端实现
│   └── types.go         # 数据结构定义和XML解析
├── README.md            # 面向人类的项目文档
└── agents.md           # 面向代理的详细指南 (本文档)
```

### 未来扩展结构规划

```
mcp/
├── main.go
├── logging.go
├── go.mod
├── go.sum
├── sources/             # 各论文源的实现目录
│   ├── arxiv/          # arXiv实现 (现有)
│   ├── semanticscholar/# Semantic Scholar
│   ├── dblp/           # DBLP
│   └── crossref/       # CrossRef
├── common/             # 共享工具和接口
│   ├── interfaces.go   # 统一的论文检索接口
│   ├── types.go        # 通用数据结构
│   └── utils.go        # 通用工具函数
├── config/             # 配置管理
    └── config.go       # 各源的配置
```

## 构建和运行

### 环境要求
- Go 1.24+
- 网络连接 (访问arXiv API)

### 快速启动
```bash
# 安装依赖
go mod tidy

# 运行服务器
go run main.go logging.go
```

### 开发模式运行
```bash
# 启用详细日志
export LOG_LEVEL=DEBUG

# 后台运行
go run main.go logging.go &
```

## 代码约定

### 命名规范
- **包名**: 全小写，使用下划线分隔 (arxiv, semantic_scholar)
- **结构体**: PascalCase (Paper, SearchResult)
- **方法**: PascalCase (SearchPapers, GetPaper)
- **变量**: camelCase (maxResults, totalCount)
- **常量**: PascalCase (ArxivAPIBaseURL, DefaultPageSize)

### 错误处理
```go
// 标准错误处理模式
if err != nil {
    log.Printf("[ERROR] 操作失败: %v", err)
    return nil, fmt.Errorf("操作失败: %w", err)
}
```

### 日志规范
```go
// 日志级别
log.Printf("[DEBUG] 详细调试信息")
log.Printf("[INFO] 重要信息")
log.Printf("[ERROR] 错误信息")
log.Printf("[FATAL] 致命错误")
```

## MCP协议集成

### 工具注册
所有MCP工具都在main.go中注册：

```go
// 添加工具示例
mcp.AddTool(server, &mcp.Tool{
    Name:        "searchArxivPapers",
    Description: "Search academic papers from arXiv...",
}, searchArxivPapers)
```

### 参数验证
```go
// 必需参数检查
if strings.TrimSpace(params.Query) == "" {
    return nil, nil, fmt.Errorf("搜索关键词不能为空")
}

// 参数范围限制
if params.MaxResults > 100 {
    params.MaxResults = 100
}
```

### 响应格式
```go
return &mcp.CallToolResult{
    Content: []mcp.Content{
        &mcp.TextContent{Text: formattedText},
    },
    Meta: map[string]interface{}{
        "structured_data": string(jsonData),
    },
}, nil, nil
```


## 扩展新论文源

### 1. 定义接口
```go
// sources/common/interfaces.go
type PaperRetriever interface {
    SearchPapers(ctx context.Context, query string, start, maxResults int) (*SearchResult, error)
    GetPaper(ctx context.Context, id string) (*Paper, error)
}
```

### 2. 实现具体源
```go
// sources/pubmed/client.go
type PubMedClient struct {
    client  *http.Client
    baseURL string
}

func (c *PubMedClient) SearchPapers(ctx context.Context, query string, start, maxResults int) (*SearchResult, error) {
    // 实现PubMed搜索逻辑
}
```

### 3. 注册到MCP
```go
// main.go 中添加
mcp.AddTool(server, &mcp.Tool{
    Name:        "searchPubMedPapers",
    Description: "Search papers from PubMed...",
}, searchPubMedPapers)
```

## 配置管理

### 环境变量
```go
// 计划支持的环境变量
ARXIV_API_TIMEOUT=30s
ARXIV_MAX_RESULTS=100
LOG_LEVEL=INFO
HTTP_PORT=8080
```

### 配置文件
```yaml
# config/sources.yaml (未来扩展)
sources:
  arxiv:
    enabled: true
    timeout: 30s
    max_results: 100
  pubmed:
    enabled: false
    api_key: "your-api-key"
```

## 性能优化

### 当前优化
- HTTP客户端连接复用
- 30秒请求超时
- XML解析优化
- 并发安全设计

### 未来优化点
- 添加Redis缓存层
- 实现请求限流
- 批量API调用优化
- 响应压缩

## 调试和监控

### 日志分析
```bash
# 查看请求日志
tail -f /var/log/mcp-server.log

# 过滤错误日志
grep "ERROR" /var/log/mcp-server.log

# 统计请求量
grep "REQUEST" /var/log/mcp-server.log | wc -l
```

### 性能监控
```go
// 添加性能指标 (未来扩展)
func recordMetrics(operation string, duration time.Duration, success bool) {
    // 记录到监控系统
}
```


### 添加新依赖
```bash
# 添加依赖
go get github.com/new/package

# 更新go.mod
go mod tidy

# 验证依赖
go mod verify
```

## 部署指南

### Docker部署
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o mcp-server main.go logging.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/mcp-server .
CMD ["./mcp-server"]
```

### 生产环境配置
```bash
# 设置生产环境变量
export LOG_LEVEL=INFO
export GOMAXPROCS=4

# 使用systemd管理
sudo systemctl enable mcp-server
sudo systemctl start mcp-server
```

## 常见问题

### Q: 如何添加新的论文检索源？
A: 参考arxiv目录的实现，创建新的源目录，实现统一的接口，然后在main.go中注册工具。

### Q: 如何处理API限流？
A: 实现请求队列、指数退避重试、重试机制。

### Q: 如何扩展数据结构？
A: 在types.go中添加新字段，确保向后兼容，使用指针类型处理可选字段。

### Q: 日志过多怎么办？
A: 实现日志级别控制，只在DEBUG模式下输出详细日志。

## 开发路线图

### Phase 1 (当前)
- ✅ arXiv基本检索功能
- ✅ MCP协议集成
- ✅ 基础日志系统

### Phase 2 (进行中)
- 🔄 添加PubMed支持
- 🔄 添加Semantic Scholar支持

### Phase 3 (规划)
- 📋 实现全文搜索
- 📋 添加论文引用分析


### 代码审查要点
- [ ] 遵循命名约定
- [ ] 有完整的错误处理
- [ ] 更新了相关文档

---

