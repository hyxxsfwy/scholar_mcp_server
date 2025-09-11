# 学术论文检索聚合 MCP 服务

基于 Go 语言实现的学术论文检索聚合 MCP (Model Context Protocol) 服务，通过统一接口同时调用多个学术数据库，提供智能去重、合并和排序的搜索结果。

## 功能特性

- 🔍 **多源聚合搜索**: 同时调用6个主要学术数据库
- 🧠 **智能去重合并**: 基于DOI和标题的智能重复检测
- 📊 **统一数据格式**: 标准化的论文元数据结构
- 🚀 **并发高性能**: 异步并行调用所有数据源
- 🎯 **精确结果**: 智能排序和相关性评分
- 🌐 **MCP 协议支持**: 完整的MCP工具接口
- 📝 **详细状态报告**: 实时数据源状态和性能监控

## 支持的数据源

- **arXiv**: 物理学、数学、计算机科学预印本 (免费)
- **Semantic Scholar**: AI驱动的学术搜索引擎 (免费)
- **Crossref**: 全球最大的DOI注册机构 (免费)
- **Scopus**: Elsevier学术数据库 (需要API密钥)
- **ADSABS**: NASA天体物理学数据系统 (需要API密钥)
- **Sci-Hub**: 学术论文PDF获取 (免费，但需注意法律风险)

## 可用工具

### 1. searchScholarPapers - 聚合学术论文搜索

从多个数据源同时搜索学术论文，自动合并去重并排序结果。

**参数:**
- `query` (string): 搜索关键词（必需）
- `author` (string): 作者筛选（可选）
- `title` (string): 标题筛选（可选）
- `journal` (string): 期刊筛选（可选）
- `year` (string): 年份筛选（可选，如"2020-2023"）
- `categories` ([]string): 分类筛选（可选）
- `min_citations` (int): 最小引用数（可选）
- `open_access_only` (bool): 仅开放获取（可选）
- `offset` (int): 偏移量，默认 0
- `limit` (int): 限制数量，默认 10，最大 100
- `sort_by` (string): 排序方式（relevance, citation_count, published_date）
- `sort_order` (string): 排序顺序（asc, desc）
- `enabled_sources` ([]string): 指定数据源（可选）

**基础搜索示例:**
```json
{
  "query": "machine learning"
}
```

**高级搜索示例:**
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

**指定数据源示例:**
```json
{
  "query": "quantum computing",
  "enabled_sources": ["arxiv", "semantic_scholar", "crossref"]
}
```

### 2. getScholarPaper - 获取论文详情

根据任意标识符获取特定论文的完整信息，自动选择最佳数据源。

**参数:**
- `identifier` (string): 论文标识符（必需）

**DOI查询示例:**
```json
{
  "identifier": "10.1038/nature12373"
}
```

**arXiv ID查询示例:**
```json
{
  "identifier": "2301.07041"
}
```

## 安装和运行

### 环境要求
- Go 1.24+
- 网络连接（访问各学术数据库API）
- 可选：Scopus API密钥
- 可选：ADSABS API密钥

### 快速启动

1. **克隆项目**
```bash
git clone https://github.com/Seelly/scholar_mcp_server.git
cd scholar_mcp_server
```

2. **安装依赖**
```bash
go mod tidy
```

3. **基础运行（使用免费数据源）**
```bash
go run main.go logging.go
```

### 完整功能运行

配置API密钥以启用全部功能：

```bash
# 设置API密钥
export SCOPUS_API_KEY="your_scopus_api_key"
export ADSABS_API_KEY="your_adsabs_api_key"

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

服务器将在 `http://127.0.0.1:8080/` 启动。

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
