package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Seelly/scholar_mcp_server/aggregator"
	"github.com/Seelly/scholar_mcp_server/common"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ScholarSearchParam 学术论文搜索参数
type ScholarSearchParam struct {
	Query          string   `json:"query"`                      // 搜索关键词
	Author         string   `json:"author,omitempty"`           // 作者筛选
	Title          string   `json:"title,omitempty"`            // 标题筛选
	Journal        string   `json:"journal,omitempty"`          // 期刊筛选
	Year           string   `json:"year,omitempty"`             // 年份筛选
	Categories     []string `json:"categories,omitempty"`       // 分类筛选
	MinCitations   int      `json:"min_citations,omitempty"`    // 最小引用数
	OpenAccessOnly bool     `json:"open_access_only,omitempty"` // 仅开放获取
	Offset         int      `json:"offset"`                     // 偏移量，默认0
	Limit          int      `json:"limit"`                      // 限制数量，默认10
	SortBy         string   `json:"sort_by,omitempty"`          // 排序方式
	SortOrder      string   `json:"sort_order,omitempty"`       // 排序顺序
	EnabledSources []string `json:"enabled_sources,omitempty"`  // 启用的数据源
}

// ScholarGetParam 获取论文详情参数
type ScholarGetParam struct {
	Identifier string `json:"identifier"` // 论文标识符 (DOI, arXiv ID, PubMed ID等)
}

// 统一的学术论文搜索功能
func searchScholarPapers(ctx context.Context, req *mcp.CallToolRequest, params *ScholarSearchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始聚合搜索学术论文 ==========")
	log.Printf("[DEBUG] 接收到的原始参数: %+v", params)

	// 转换为通用搜索参数并标准化
	searchParams := common.SearchParams{
		Query:          params.Query,
		Author:         params.Author,
		Title:          params.Title,
		Journal:        params.Journal,
		Year:           params.Year,
		Categories:     params.Categories,
		MinCitations:   params.MinCitations,
		OpenAccessOnly: params.OpenAccessOnly,
		Offset:         params.Offset,
		Limit:          params.Limit,
		SortBy:         params.SortBy,
		SortOrder:      params.SortOrder,
	}
	common.NormalizeSearchParams(&searchParams)

	// 设置默认值
	if searchParams.Limit > 100 {
		searchParams.Limit = 100
	}

	// 同步标准化后的显示参数
	params.Query = searchParams.Query
	params.Offset = searchParams.Offset
	params.Limit = searchParams.Limit

	// 验证搜索条件
	if !common.HasSearchTerms(searchParams) {
		log.Printf("[ERROR] 搜索条件为空，终止请求")
		return nil, nil, fmt.Errorf("搜索条件不能为空，至少填写一个搜索词或筛选字段")
	}

	log.Printf("[INFO] 开始聚合搜索: 关键词='%s', Offset=%d, Limit=%d", searchParams.Query, searchParams.Offset, searchParams.Limit)

	// 创建聚合器并搜索
	agg := aggregator.NewScholarAggregator()
	result, err := agg.SearchPapers(ctx, searchParams)
	if err != nil {
		log.Printf("[ERROR] 聚合搜索失败: %v", err)
		return nil, nil, fmt.Errorf("搜索失败: %w", err)
	}

	// 格式化输出
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("🔬 学术论文聚合搜索结果 (关键词: '%s')\n", params.Query))
	resultBuilder.WriteString(fmt.Sprintf("数据源: %d个总计, %d个活跃\n", result.TotalSources, result.ActiveSources))
	resultBuilder.WriteString(fmt.Sprintf("搜索耗时: %v\n", result.SearchTime))
	resultBuilder.WriteString(fmt.Sprintf("显示第%d-%d条结果，共找到约%d篇论文\n\n", params.Offset+1, params.Offset+len(result.Papers), result.TotalResults))

	for i, paper := range result.Papers {
		resultBuilder.WriteString(fmt.Sprintf("📄 %d. %s\n", i+1, paper.Title))
		resultBuilder.WriteString(fmt.Sprintf("   来源: %s\n", paper.Source))
		if paper.ID != "" {
			resultBuilder.WriteString(fmt.Sprintf("   ID: %s\n", paper.ID))
		}
		if paper.DOI != "" {
			resultBuilder.WriteString(fmt.Sprintf("   DOI: %s\n", paper.DOI))
		}

		if len(paper.Authors) > 0 {
			authors := strings.Join(paper.Authors, ", ")
			if len(authors) > 100 {
				authors = authors[:100] + "..."
			}
			resultBuilder.WriteString(fmt.Sprintf("   作者: %s\n", authors))
		}

		if paper.Journal != "" {
			resultBuilder.WriteString(fmt.Sprintf("   期刊: %s\n", paper.Journal))
		}
		if paper.PublishedDate != "" {
			resultBuilder.WriteString(fmt.Sprintf("   发表日期: %s\n", paper.PublishedDate))
		}
		if paper.CitationCount > 0 {
			resultBuilder.WriteString(fmt.Sprintf("   引用数: %d\n", paper.CitationCount))
		}
		if paper.IsOpenAccess {
			resultBuilder.WriteString("   📖 开放获取\n")
		}
		if paper.Abstract != "" {
			abstract := paper.Abstract
			if len(abstract) > 300 {
				abstract = abstract[:300] + "..."
			}
			resultBuilder.WriteString(fmt.Sprintf("   摘要: %s\n", abstract))
		}
		if paper.URL != "" {
			resultBuilder.WriteString(fmt.Sprintf("   链接: %s\n", paper.URL))
		}
		if paper.PDFURL != "" {
			resultBuilder.WriteString(fmt.Sprintf("   PDF: %s\n", paper.PDFURL))
		}
		resultBuilder.WriteString("\n")
	}

	// 添加数据源状态信息
	resultBuilder.WriteString("📊 数据源状态:\n")
	for _, status := range result.SourceStatus {
		statusIcon := "❌"
		if status.Available {
			statusIcon = "✅"
		}
		resultBuilder.WriteString(fmt.Sprintf("   %s %s: %s (%dms)\n",
			statusIcon, status.Source, status.Message, status.ResponseTime))
	}

	// 添加搜索建议
	if len(result.Suggestions) > 0 {
		resultBuilder.WriteString("\n💡 搜索建议:\n")
		for _, hint := range result.Suggestions {
			resultBuilder.WriteString(fmt.Sprintf("   - %s: %s\n", hint.Type, hint.Description))
		}
	}

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] ========== 聚合搜索完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// 获取学术论文详情功能
func getScholarPaper(ctx context.Context, req *mcp.CallToolRequest, params *ScholarGetParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始获取学术论文详情 ==========")
	log.Printf("[DEBUG] 接收到的参数: %+v", params)

	// 验证标识符参数
	if strings.TrimSpace(params.Identifier) == "" {
		log.Printf("[ERROR] 论文标识符为空，终止请求")
		return nil, nil, fmt.Errorf("论文标识符不能为空")
	}

	log.Printf("[INFO] 开始获取论文详情: 标识符='%s'", params.Identifier)

	// 创建聚合器并获取论文
	agg := aggregator.NewScholarAggregator()
	paper, err := agg.GetPaper(ctx, params.Identifier)
	if err != nil {
		log.Printf("[ERROR] 获取论文详情失败: %v", err)
		return nil, nil, fmt.Errorf("获取论文详情失败: %w", err)
	}

	// 格式化输出
	var resultBuilder strings.Builder
	resultBuilder.WriteString("📄 学术论文详情\n\n")
	resultBuilder.WriteString(fmt.Sprintf("📝 标题: %s\n\n", paper.Title))
	resultBuilder.WriteString(fmt.Sprintf("🗃️ 数据源: %s\n", paper.Source))

	if paper.ID != "" {
		resultBuilder.WriteString(fmt.Sprintf("🆔 ID: %s\n", paper.ID))
	}
	if paper.DOI != "" {
		resultBuilder.WriteString(fmt.Sprintf("🔗 DOI: %s\n", paper.DOI))
	}

	if len(paper.Authors) > 0 {
		resultBuilder.WriteString(fmt.Sprintf("👥 作者: %s\n\n", strings.Join(paper.Authors, ", ")))
	}

	if paper.Journal != "" {
		resultBuilder.WriteString(fmt.Sprintf("📍 期刊: %s\n", paper.Journal))
	}
	if paper.Publisher != "" {
		resultBuilder.WriteString(fmt.Sprintf("🏢 出版商: %s\n", paper.Publisher))
	}
	if paper.PublishedDate != "" {
		resultBuilder.WriteString(fmt.Sprintf("📅 发表日期: %s\n", paper.PublishedDate))
	}
	if paper.Volume != "" {
		resultBuilder.WriteString(fmt.Sprintf("📖 卷号: %s\n", paper.Volume))
	}
	if paper.Issue != "" {
		resultBuilder.WriteString(fmt.Sprintf("📄 期号: %s\n", paper.Issue))
	}
	if paper.Pages != "" {
		resultBuilder.WriteString(fmt.Sprintf("📄 页码: %s\n", paper.Pages))
	}

	if paper.CitationCount > 0 {
		resultBuilder.WriteString(fmt.Sprintf("📊 引用数: %d\n", paper.CitationCount))
	}
	if paper.ReadCount > 0 {
		resultBuilder.WriteString(fmt.Sprintf("👁️ 阅读数: %d\n", paper.ReadCount))
	}

	if paper.IsOpenAccess {
		resultBuilder.WriteString("📖 开放获取: 是\n")
	}

	if len(paper.Categories) > 0 {
		resultBuilder.WriteString(fmt.Sprintf("🏷️ 分类: %s\n", strings.Join(paper.Categories, ", ")))
	}

	if paper.Language != "" {
		resultBuilder.WriteString(fmt.Sprintf("🌐 语言: %s\n", paper.Language))
	}

	if paper.Type != "" {
		resultBuilder.WriteString(fmt.Sprintf("📋 类型: %s\n", paper.Type))
	}

	if paper.Abstract != "" {
		resultBuilder.WriteString(fmt.Sprintf("\n📋 摘要:\n%s\n", paper.Abstract))
	}

	// 外部链接
	if len(paper.ExternalIDs) > 0 {
		resultBuilder.WriteString("\n🔗 外部链接:\n")
		for source, id := range paper.ExternalIDs {
			if id != "" {
				resultBuilder.WriteString(fmt.Sprintf("   %s: %s\n", source, id))
			}
		}
	}

	if paper.URL != "" {
		resultBuilder.WriteString(fmt.Sprintf("\n🌐 原始链接: %s\n", paper.URL))
	}
	if paper.PDFURL != "" {
		resultBuilder.WriteString(fmt.Sprintf("📄 PDF下载: %s\n", paper.PDFURL))
	}

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(paper)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] ========== 学术论文详情获取完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

func main() {
	httpMode := flag.Bool("http", false, "Run as HTTP server instead of stdio transport")
	port := flag.String("port", "8899", "HTTP server port (only used with --http)")
	flag.Parse()

	if *httpMode {
		runHTTPServer(*port)
	} else {
		runStdioServer()
	}
}

func createServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "scholar-aggregator-mcp-server",
		Version: "2.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "searchScholarPapers",
		Description: "Search academic papers from multiple sources simultaneously (arXiv, Semantic Scholar, Crossref, Scopus, ADSABS, Sci-Hub). Automatically aggregates, deduplicates, and ranks results. Supports advanced filtering by author, journal, year, citations, and open access status. Returns unified paper metadata with source attribution.",
	}, searchScholarPapers)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "getScholarPaper",
		Description: "Get detailed information of a specific academic paper by any identifier (DOI, arXiv ID, PubMed ID, Semantic Scholar ID, etc.). Automatically determines the best data source based on identifier type and retrieves comprehensive metadata including citations, abstracts, and external links.",
	}, getScholarPaper)

	return server
}

func runStdioServer() {
	// Redirect log output to stderr so it doesn't interfere with MCP protocol on stdout.
	log.SetOutput(os.Stderr)

	log.Printf("[INFO] ========== 学术论文检索聚合MCP服务器启动 (stdio) ==========")
	server := createServer()
	log.Printf("[INFO] 统一学术论文搜索工具已注册 (searchScholarPapers, getScholarPaper)")

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func runHTTPServer(port string) {
	log.Printf("[INFO] ========== 学术论文检索聚合MCP服务器启动 (HTTP) ==========")

	server := createServer()

	mcpHandler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		log.Printf("[DEBUG] HTTP请求处理: %s %s", req.Method, req.URL.Path)
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true})

	// VSCode probes /.well-known/oauth-protected-resource before sending MCP
	// requests. If that path reaches the MCP handler it returns 400, which
	// triggers VSCode's OAuth flow and blocks the connection. Return 404 here
	// so VSCode skips OAuth and connects directly.
	router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/.well-known/") {
			http.NotFound(w, r)
			return
		}
		mcpHandler.ServeHTTP(w, r)
	})

	rootHandler := loggingHandler(router)

	log.Printf("[INFO] MCP服务端点: http://0.0.0.0:%s", port)
	log.Printf("[INFO] 统一工具接口: searchScholarPapers, getScholarPaper")
	log.Printf("[INFO] 支持的数据源: arXiv, Semantic Scholar, Crossref, Scopus, ADSABS, Sci-Hub")
	log.Printf("[INFO] ========== 服务器启动完成，等待连接 ==========")
	if err := http.ListenAndServe(":"+port, rootHandler); err != nil {
		log.Fatalf("[FATAL] 服务器启动失败: %v", err)
	}
}
