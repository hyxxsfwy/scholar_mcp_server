# arXiv论文检索 MCP 服务

基于 Go 语言实现的 arXiv 论文检索 MCP (Model Context Protocol) 服务，提供强大的学术论文搜索和获取功能。

## 功能特性

- 🔍 **arXiv论文搜索**: 支持关键词、分类、作者等多种搜索方式
- 📄 **论文详情获取**: 根据arXiv ID获取完整的论文元数据
- 🌐 **MCP 协议支持**: 完全兼容 MCP 协议，可与 Claude Desktop 等客户端集成
- 🚀 **高性能**: 使用 Go 语言实现，支持并发处理
- 📊 **结构化数据**: 返回JSON格式的结构化论文数据
- 🎯 **准确性**: 直接调用官方arXiv API，数据准确可靠

## 可用工具

### 1. searchArxivPapers - arXiv论文搜索

搜索arXiv上的学术论文。

**参数:**
- `query` (string): 搜索关键词（必需）
  - 支持简单关键词: "machine learning"
  - 支持分类搜索: "cat:cs.AI"
  - 支持作者搜索: "au:John Doe"
  - 支持组合查询: "machine learning AND cat:cs.LG"
- `start` (int): 开始位置，默认 0
- `max_results` (int): 最大结果数，默认 10，最大 100
- `sort_by` (string): 排序方式（可选）
- `sort_order` (string): 排序顺序（可选）

**示例:**
```json
{
  "query": "quantum computing",
  "start": 0,
  "max_results": 10
}
```

### 2. getArxivPaper - 获取论文详情

根据arXiv ID获取特定论文的详细信息。

**参数:**
- `id` (string): arXiv ID（必需），例如 "2301.07041"

**示例:**
```json
{
  "id": "2301.07041"
}
```

## 安装和运行

### 1. 克隆项目
```bash
cd /Users/mwh/GolandProjects/mcp
```

### 2. 安装依赖
```bash
go mod tidy
```

### 3. 运行服务器
```bash
go run main.go logging.go
```

服务器将在 `http://127.0.0.1:8080` 启动。

## MCP 客户端配置

### Claude Desktop 配置

在 `~/.cursor/mcp.json` 或 Claude Desktop 配置文件中添加:

```json
{
  "mcpServers": {
    "arXiv论文检索服务": {
      "url": "http://127.0.0.1:8080/mcp"
    }
  }
}
```

## 使用示例

### 搜索论文
```bash
# 使用 curl 测试搜索功能
curl -X POST http://127.0.0.1:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "searchArxivPapers",
      "arguments": {
        "query": "deep learning",
        "max_results": 5
      }
    }
  }'
```

### 获取论文详情
```bash
curl -X POST http://127.0.0.1:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "getArxivPaper",
      "arguments": {
        "id": "2301.07041"
      }
    }
  }'
```

## 数据结构

### Paper (论文信息)
```go
type Paper struct {
    ID         string   `json:"id"`          // arXiv ID
    Title      string   `json:"title"`       // 论文标题
    Authors    []string `json:"authors"`     // 作者列表
    Abstract   string   `json:"abstract"`    // 摘要
    Categories []string `json:"categories"`  // 分类
    Published  string   `json:"published"`   // 发布日期
    Updated    string   `json:"updated"`     // 更新日期
    URL        string   `json:"url"`         // arXiv URL
    PDFURL     string   `json:"pdf_url"`     // PDF下载链接
    DOI        string   `json:"doi"`         // DOI
    JournalRef string   `json:"journal_ref"` // 期刊引用
    Comment    string   `json:"comment"`     // 注释
}
```

### SearchResult (搜索结果)
```go
type SearchResult struct {
    Query      string  `json:"query"`       // 查询关键词
    Start      int     `json:"start"`       // 开始位置
    MaxResults int     `json:"max_results"` // 最大结果数
    TotalCount int     `json:"total_count"` // 总数量
    Papers     []Paper `json:"papers"`      // 论文列表
}
```

## arXiv API 特性

### 搜索语法

arXiv 支持强大的搜索语法：

- **基本搜索**: `machine learning`
- **分类搜索**: `cat:cs.AI` (计算机科学-人工智能)
- **作者搜索**: `au:John Doe`
- **标题搜索**: `ti:quantum`
- **摘要搜索**: `abs:neural network`
- **组合搜索**: `machine learning AND cat:cs.LG`
- **日期范围**: `submittedDate:[20220101 TO 20221231]`

### 主要分类

- `cs.AI` - 人工智能
- `cs.LG` - 机器学习
- `cs.CV` - 计算机视觉
- `cs.CL` - 计算语言学
- `physics.gen-ph` - 一般物理
- `math.NA` - 数值分析
- `stat.ML` - 统计机器学习

## 技术实现

- **HTTP 客户端**: 使用 `net/http` 包调用arXiv API
- **XML 解析**: 使用 `encoding/xml` 解析arXiv返回的Atom格式数据
- **MCP 协议**: 基于 `github.com/modelcontextprotocol/go-sdk/mcp`
- **并发安全**: 支持多个并发请求
- **错误处理**: 完善的错误处理和日志记录

## 性能优化

- 默认30秒请求超时
- 支持分页查询避免一次性加载过多数据
- 最大单次查询100篇论文的限制
- 高效的XML解析和数据转换

## 扩展功能

可以考虑添加的功能:
- 论文缓存机制
- 更复杂的搜索过滤器
- 论文引用分析
- 本地PDF下载和管理
- 与其他学术数据库的集成

## 许可证

MIT License
