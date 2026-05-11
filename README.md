# 学术论文检索聚合 MCP 服务

基于 Go 语言实现的学术论文检索聚合 MCP (Model Context Protocol) 服务，通过统一接口同时调用多个学术数据库，提供智能去重、合并和排序的搜索结果。

## 功能特性

- 🔍 **多源聚合搜索**: 同时调用多个主要学术数据库
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
- **OpenAlex**: 开放学术图谱，提供论文元数据、引用数与开放获取链接 (免费，默认启用)
- **Scopus**: Elsevier学术数据库 (需要API密钥)
- **ADSABS**: NASA天体物理学数据系统 (需要API密钥)
- **Google Scholar**: 通过 SerpAPI 兼容第三方 SERP API 接入 (需要API密钥，默认关闭)
- **券商金工研报**: 通过用户授权的 JSON/RSS/Atom Feed 接入券商/研报平台数据 (默认关闭)
- **Sci-Hub**: 学术论文PDF获取 (免费，但需注意法律风险)

## 可用工具

### 1. searchScholarPapers - 聚合学术论文搜索

从多个数据源同时搜索学术论文，自动合并去重并排序结果。

**参数:**
- `query` (string): 搜索关键词（可选；至少填写一个搜索词或筛选字段）
- `author` (string): 作者筛选（可选）
- `title` (string): 标题筛选（可选）
- `abstract` (string): 摘要筛选（可选）
- `journal` (string): 期刊筛选（可选）
- `publisher` (string): 出版商/机构筛选（可选，券商研报可填写券商名称）
- `year` (string): 年份筛选（可选，如"2020-2023"）
- `year_range` (string): 年份范围筛选（可选，如"2020-2023"）
- `language` (string): 语言筛选（可选）
- `type` (string): 文档类型筛选（可选，如"research-report"）
- `categories` ([]string): 分类筛选（可选）
- `min_citations` (int): 最小引用数（可选）
- `open_access_only` (bool): 仅开放获取（可选）
- `offset` (int): 偏移量，默认 0
- `limit` (int): 限制数量，默认 10，最大 100
- `sort_by` (string): 排序方式（relevance, citation_count, published_date, read_count, score）
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

### 配置文件

项目根目录提供 [config.jsonc](config.jsonc)，支持 `//`、`/* ... */` 注释和尾随逗号。程序会先读取环境变量，再用 `config.jsonc` 中实际写出的字段覆盖环境变量；未写出的字段仍会回退到环境变量或内置默认值。

配置文件包含四类配置：

- `server`: 默认运行模式和 HTTP 端口。命令行参数 `--http`、`--port` 会覆盖这里的默认值。
- `request`: 全局请求超时、重试次数和可选统一限速。
- `enrichment`: 搜索结果增强。启用后会对当前返回结果中排名前 `top_n` 的文献并行生成 LLM 大纲/梗概。
- `sources`: 各数据源的启用状态、API key、邮箱、base URL、限速，以及券商研报 feed。

常用配置示例：

```jsonc
{
  "server": {
    "http": true,
    "port": "8899"
  },
  "request": {
    "timeout": 30,
    "max_retries": 3
  },
  "enrichment": {
    "enabled": true,
    "top_n": 5,
    "prefetch_pdf": true,
    "pdf_fallback": "open_access",
    "max_pdf_bytes": 10485760,
    "pdf_text_max_chars": 12000,
    "max_input_chars": 16000,
    "timeout": 60,
    "llm": {
      "api_key": "your_deepseek_or_openai_compatible_api_key",
      "base_url": "https://api.deepseek.com/v1",
      "model": "deepseek-v4-flash",
      "max_tokens": 1200,
      "temperature": 0.2,
      "max_concurrency": 5
    }
  },
  "sources": {
    "openalex": {
      "enabled": true,
      "email": "your_email@example.com"
    },
    "google_scholar": {
      "enabled": true,
      "api_key": "your_serpapi_api_key",
      "base_url": "https://serpapi.com/search.json"
    },
    "broker_research": {
      "enabled": true,
      "api_key": "your_authorized_feed_token",
      "feeds": [
        {
          "name": "华泰证券金工",
          "broker": "华泰证券",
          "team": "金融工程",
          "format": "json",
          "url": "https://example.com/htsc/quant-reports.json?q={query}&limit={limit}"
        },
        {
          "name": "中信证券金工RSS",
          "broker": "中信证券",
          "team": "金融工程",
          "format": "rss",
          "url": "https://example.com/citic/quant/rss.xml"
        }
      ]
    }
  }
}
```

`enrichment.prefetch_pdf` 会优先下载搜索结果中已有的 HTTP(S) `pdf_url`。当 `pdf_fallback` 为 `open_access` 时，如果结果没有 PDF 或下载失败，系统会用 DOI 查询 OpenAlex 的开放获取位置并尝试下载合法 OA PDF；设为 `none` 则只使用结果自带 PDF。该自动链路不会调用 Sci-Hub。启用该功能时，摘要和 PDF 文本节选会发送到你配置的外部 LLM API；如果不希望传输 PDF 内容，可以将 `prefetch_pdf` 设为 `false`，系统仍会基于已有摘要生成大纲/梗概。

兼容的环境变量仍然可用：`REQUEST_TIMEOUT`、`SCOPUS_API_KEY`、`ADSABS_API_KEY`、`OPENALEX_EMAIL`、`CROSSREF_EMAIL`、`SERPAPI_API_KEY`、`GOOGLE_SCHOLAR_API_KEY`、`BROKER_RESEARCH_API_KEY`、`BROKER_RESEARCH_FEEDS`，以及各 `ENABLE_*` 开关。结果增强也支持 `ENABLE_RESULT_ENRICHMENT`、`RESULT_ENRICHMENT_TOP_N`、`RESULT_ENRICHMENT_PREFETCH_PDF`、`RESULT_ENRICHMENT_PDF_FALLBACK`、`RESULT_ENRICHMENT_LLM_API_KEY`、`LLM_API_KEY`、`DEEPSEEK_API_KEY`、`RESULT_ENRICHMENT_LLM_BASE_URL`、`RESULT_ENRICHMENT_LLM_MODEL` 等环境变量。只要同名配置字段写在 [config.jsonc](config.jsonc) 中，配置文件优先。

### 完整功能运行

编辑 [config.jsonc](config.jsonc) 后运行服务器：

```bash
go run main.go logging.go
```

### 生产部署

```bash
# 编译二进制文件
go build -o scholar_mcp_server .

# 后台运行
nohup ./scholar_mcp_server --http > server.log 2>&1 &
```

服务器将在 `http://0.0.0.0:8899/` 启动。

## MCP 客户端配置

### Claude Desktop 配置

在 `~/.cursor/mcp.json` 或 Claude Desktop 配置文件中添加:

```json
{
  "mcpServers": {
    "arXiv论文检索服务": {
      "url": "http://localhost:8899/"
    }
  }
}
```

## 使用示例

### 搜索论文
```bash
# 使用 curl 测试搜索功能
curl -X POST http://localhost:8899/ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "searchScholarPapers",
      "arguments": {
        "query": "deep learning",
        "limit": 5
      }
    }
  }'
```

### 获取论文详情
```bash
curl -X POST http://localhost:8899/ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "getScholarPaper",
      "arguments": {
        "identifier": "2301.07041"
      }
    }
  }'
```

