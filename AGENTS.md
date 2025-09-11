# 学术论文检索聚合服务 - 代理指南

本文档为编程代理提供详细的项目信息，包括构建步骤、代码结构、测试约定和扩展指南。

## 项目概述

这是一个基于Go语言实现的MCP (Model Context Protocol) 学术论文检索聚合服务，通过统一接口同时调用多个学术数据库，提供智能去重、合并和排序的搜索结果。

### 核心功能
- 🔍 **多源聚合搜索**: 同时调用6个主要学术数据库
- 🧠 **智能去重合并**: 基于DOI和标题的智能重复检测
- 📊 **统一数据格式**: 标准化的论文元数据结构
- 🚀 **并发高性能**: 异步并行调用所有数据源
- 🎯 **精确结果**: 智能排序和相关性评分
- 🌐 **MCP协议支持**: 完整的MCP工具接口
- 📝 **详细状态报告**: 实时数据源状态和性能监控

### 支持的数据源
- **arXiv**: 物理学、数学、计算机科学预印本 (免费)
- **Semantic Scholar**: AI驱动的学术搜索引擎 (免费)
- **Crossref**: 全球最大的DOI注册机构 (免费)
- **Scopus**: Elsevier学术数据库 (需要API密钥)
- **ADSABS**: NASA天体物理学数据系统 (需要API密钥)
- **Sci-Hub**: 学术论文PDF获取 (免费，但需注意法律风险)

### 技术栈
- **Go**: 1.24+
- **MCP SDK**: github.com/modelcontextprotocol/go-sdk v0.3.1
- **并发处理**: sync包，goroutines
- **HTTP客户端**: net/http (标准库)
- **数据解析**: encoding/json, encoding/xml

## 项目结构

```
scholar_mcp_server/
├── main.go              # 统一MCP服务器入口和聚合工具接口
├── logging.go           # HTTP请求日志中间件
├── go.mod               # Go模块定义 (github.com/Seelly/scholar_mcp_server)
├── go.sum               # 依赖校验文件
├── aggregator/          # 核心聚合器模块
│   ├── aggregator.go    # 主聚合器实现，并发调用和结果合并
│   └── sources.go       # 各数据源的适配器实现
├── common/              # 通用工具和接口
│   ├── types.go         # 统一数据结构和接口定义
│   └── utils.go         # 通用工具函数 (验证、清理、去重等)
├── arxiv/               # arXiv数据源实现
│   ├── client.go        # arXiv API客户端
│   └── types.go         # arXiv特定数据结构
├── semanticscholar/     # Semantic Scholar数据源实现
│   ├── client.go        # Semantic Scholar API客户端
│   └── types.go         # Semantic Scholar数据结构
├── crossref/            # Crossref数据源实现
│   ├── client.go        # Crossref API客户端
│   └── types.go         # Crossref数据结构
├── scopus/              # Scopus数据源实现 (需要API密钥)
│   ├── client.go        # Scopus API客户端
│   └── types.go         # Scopus数据结构
├── adsabs/              # ADSABS数据源实现 (需要API密钥)
│   ├── client.go        # ADSABS API客户端
│   └── types.go         # ADSABS数据结构
├── scihub/              # Sci-Hub数据源实现
│   ├── client.go        # Sci-Hub客户端
│   └── types.go         # Sci-Hub数据结构
├── README.md            # 面向用户的项目文档
└── agents.md           # 面向代理的详细指南 (本文档)
```

### 架构设计

#### 核心架构
```
用户请求 → MCP接口 → 聚合器 → 并发调用各数据源 → 结果合并去重 → 统一响应
```

#### 数据流
1. **请求接收**: main.go接收MCP工具调用
2. **参数验证**: common/utils.go验证和标准化参数
3. **并发搜索**: aggregator.go并发调用所有启用的数据源
4. **结果适配**: sources.go将各源数据转换为统一格式
5. **智能合并**: 基于DOI和标题进行去重和合并
6. **排序输出**: 按相关性、引用数等排序返回

## 构建和运行

### 环境要求
- Go 1.24+
- 网络连接 (访问各学术数据库API)
- 可选: Scopus API密钥
- 可选: ADSABS API密钥

### 快速启动
```bash
# 克隆项目
git clone https://github.com/Seelly/scholar_mcp_server.git
cd scholar_mcp_server

# 安装依赖
go mod tidy

# 基础运行 (使用免费数据源)
go run main.go logging.go
```

### 完整功能运行
```bash
# 配置API密钥以启用完整功能
export SCOPUS_API_KEY="your_scopus_api_key"
export ADSABS_API_KEY="your_adsabs_api_key"

# 启用详细日志
export LOG_LEVEL=DEBUG

# 运行服务器
go run main.go logging.go
```

### 生产部署
```bash
# 编译二进制文件
go build -o scholar-server main.go logging.go

# 后台运行
nohup ./scholar-server > server.log 2>&1 &
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

### 统一工具接口
服务器提供两个核心MCP工具：

```go
// 1. 聚合搜索工具
mcp.AddTool(server, &mcp.Tool{
    Name:        "searchScholarPapers",
    Description: "Search academic papers from multiple sources simultaneously...",
}, searchScholarPapers)

// 2. 论文详情获取工具
mcp.AddTool(server, &mcp.Tool{
    Name:        "getScholarPaper", 
    Description: "Get detailed information of a specific academic paper...",
}, getScholarPaper)
```

### 搜索参数结构
```go
type ScholarSearchParam struct {
    Query             string   `json:"query"`              // 必需: 搜索关键词
    Author            string   `json:"author,omitempty"`   // 可选: 作者筛选
    Title             string   `json:"title,omitempty"`    // 可选: 标题筛选
    Journal           string   `json:"journal,omitempty"`  // 可选: 期刊筛选
    Year              string   `json:"year,omitempty"`     // 可选: 年份筛选
    Categories        []string `json:"categories,omitempty"` // 可选: 分类筛选
    MinCitations      int      `json:"min_citations,omitempty"` // 可选: 最小引用数
    OpenAccessOnly    bool     `json:"open_access_only,omitempty"` // 可选: 仅开放获取
    Offset            int      `json:"offset"`             // 可选: 偏移量，默认0
    Limit             int      `json:"limit"`              // 可选: 限制数量，默认10
    SortBy            string   `json:"sort_by,omitempty"`  // 可选: 排序方式
    SortOrder         string   `json:"sort_order,omitempty"` // 可选: 排序顺序
    EnabledSources    []string `json:"enabled_sources,omitempty"` // 可选: 指定数据源
}
```

### 聚合响应格式
```go
return &mcp.CallToolResult{
    Content: []mcp.Content{
        &mcp.TextContent{Text: formattedAggregatedResults},
    },
    Meta: map[string]interface{}{
        "structured_data": aggregatedSearchResult, // 包含所有源的结果
        "source_status":   sourceStatusInfo,       // 各源状态信息
        "search_time":     totalSearchTime,        // 总搜索时间
        "dedup_info":      deduplicationStats,     // 去重统计
    },
}, nil, nil
```

### 使用示例

#### 基础搜索
```json
{
  "query": "machine learning"
}
```

#### 高级搜索
```json
{
  "query": "deep learning",
  "author": "Hinton", 
  "year": "2020-2023",
  "min_citations": 100,
  "open_access_only": true,
  "limit": 20,
  "sort_by": "citation_count",
  "sort_order": "desc"
}
```

#### 指定数据源
```json
{
  "query": "quantum computing",
  "enabled_sources": ["arxiv", "semantic_scholar", "crossref"]
}
```

#### 获取论文详情
```json
{
  "identifier": "10.1038/nature12373"  // DOI
}
```

```json
{
  "identifier": "2301.07041"  // arXiv ID
}
```


## 扩展新论文源

### 1. 创建数据源目录
```bash
mkdir newsource
cd newsource
```

### 2. 实现数据源类型定义
```go
// newsource/types.go
package newsource

type Paper struct {
    // 定义该数据源特有的字段
    ID       string `json:"id"`
    Title    string `json:"title"`
    // ... 其他字段
}

type SearchResult struct {
    Papers []Paper `json:"papers"`
    Total  int     `json:"total"`
}

type SearchParam struct {
    Query  string `json:"query"`
    Offset int    `json:"offset"`
    Limit  int    `json:"limit"`
}
```

### 3. 实现客户端
```go
// newsource/client.go
package newsource

func NewClient() *Client {
    return &Client{
        client: &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *Client) SearchPapers(ctx context.Context, query string, offset, limit int) (*SearchResult, error) {
    // 实现搜索逻辑
}

func (c *Client) GetPaper(ctx context.Context, id string) (*Paper, error) {
    // 实现获取详情逻辑
}
```

### 4. 在聚合器中集成
```go
// aggregator/sources.go 中添加
func (a *ScholarAggregator) searchNewSource(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
    client := newsource.NewClient()
    result, err := client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
    if err != nil {
        return nil, 0, err
    }

    // 转换为统一格式
    var papers []common.UnifiedPaper
    for _, paper := range result.Papers {
        unified := common.UnifiedPaper{
            ID:     paper.ID,
            Title:  paper.Title,
            Source: "newsource",
            // ... 映射其他字段
        }
        papers = append(papers, unified)
    }
    
    return papers, result.Total, nil
}
```

### 5. 启用新数据源
```go
// aggregator/aggregator.go 中添加并发调用
if a.config.EnableNewSource {
    totalSources++
    wg.Add(1)
    go func() {
        defer wg.Done()
        start := time.Now()
        papers, total, err := a.searchNewSource(ctx, params)
        // ... 处理结果
    }()
}
```

## 配置管理

### 环境变量
```bash
# API密钥配置
export SCOPUS_API_KEY="your_scopus_api_key"
export ADSABS_API_KEY="your_adsabs_api_key"

# 系统配置
export LOG_LEVEL="INFO"                    # DEBUG, INFO, ERROR
export HTTP_PORT="8080"                    # 服务端口
export REQUEST_TIMEOUT="30s"               # API请求超时
export MAX_CONCURRENT_SEARCHES="10"        # 最大并发搜索数
export ENABLE_CACHE="false"                # 是否启用结果缓存
export CACHE_EXPIRATION="300s"             # 缓存过期时间

# 数据源控制
export ENABLE_ARXIV="true"                 # 是否启用arXiv
export ENABLE_SEMANTIC_SCHOLAR="true"      # 是否启用Semantic Scholar
export ENABLE_CROSSREF="true"              # 是否启用Crossref
export ENABLE_SCOPUS="false"               # 是否启用Scopus (需要API密钥)
export ENABLE_ADSABS="false"               # 是否启用ADSABS (需要API密钥)
export ENABLE_SCIHUB="true"                # 是否启用Sci-Hub
```

### 配置结构
```go
// common/types.go
type Config struct {
    // API密钥
    ScopusAPIKey          string
    AdsabsAPIKey          string
    
    // 性能配置
    RequestTimeout        int
    MaxRetries            int
    MaxConcurrentSearches int
    
    // 缓存配置
    EnableCache           bool
    CacheExpiration       int
    
    // 数据源开关
    EnableArxiv           bool
    EnableSemanticScholar bool
    EnableCrossref        bool
    EnableScopus          bool
    EnableAdsabs          bool
    EnableSciHub          bool
    
    // 搜索限制
    DefaultLimit          int
    MaxLimit              int
    DefaultSort           string
}
```

## 性能优化

### 当前优化
- **并发搜索**: 所有数据源并行调用，大幅减少总响应时间
- **智能去重**: 基于DOI和标题的高效去重算法
- **连接复用**: HTTP客户端连接池复用
- **超时控制**: 30秒API请求超时，避免长时间等待
- **内存优化**: 流式处理，避免大量数据积累
- **错误隔离**: 单个数据源失败不影响其他源

### 性能指标
- **并发数**: 最多6个数据源同时搜索
- **平均响应时间**: 3-8秒 (取决于启用的数据源数量)
- **内存使用**: < 100MB (典型搜索)
- **CPU使用**: 低CPU占用，主要为网络I/O等待

### 扩展优化
- **缓存层**: 可配置的内存缓存，减少重复请求
- **请求限流**: 避免API配额超限
- **结果预处理**: 后台预热热门查询结果
- **负载均衡**: 支持多实例部署

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

