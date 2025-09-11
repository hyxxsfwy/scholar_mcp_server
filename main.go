package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	// 设置默认值
	if params.Offset < 0 {
		params.Offset = 0
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	// 验证查询参数
	if strings.TrimSpace(params.Query) == "" {
		log.Printf("[ERROR] 搜索关键词为空，终止请求")
		return nil, nil, fmt.Errorf("搜索关键词不能为空")
	}

	log.Printf("[INFO] 开始聚合搜索: 关键词='%s', Offset=%d, Limit=%d", params.Query, params.Offset, params.Limit)

	// 转换为通用搜索参数
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
	log.Printf("[INFO] ========== 学术论文检索聚合MCP服务器启动 ==========")
	log.Printf("[INFO] 正在初始化统一学术论文检索MCP服务器...")

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "scholar-aggregator-mcp-server",
		Version: "2.0.0",
	}, nil)
	log.Printf("[DEBUG] MCP服务器创建完成，名称: %s, 版本: %s", "scholar-aggregator-mcp-server", "2.0.0")

	// 添加统一的学术论文搜索工具
	log.Printf("[DEBUG] 正在注册统一学术论文搜索工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "searchScholarPapers",
		Description: "Search academic papers from multiple sources simultaneously (arXiv, Semantic Scholar, Crossref, Scopus, ADSABS, Sci-Hub). Automatically aggregates, deduplicates, and ranks results. Supports advanced filtering by author, journal, year, citations, and open access status. Returns unified paper metadata with source attribution.",
	}, searchScholarPapers)
	log.Printf("[INFO] 统一学术论文搜索工具注册完成")

	// 添加获取学术论文详情工具
	log.Printf("[DEBUG] 正在注册学术论文详情获取工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "getScholarPaper",
		Description: "Get detailed information of a specific academic paper by any identifier (DOI, arXiv ID, PubMed ID, Semantic Scholar ID, etc.). Automatically determines the best data source based on identifier type and retrieves comprehensive metadata including citations, abstracts, and external links.",
	}, getScholarPaper)
	log.Printf("[INFO] 学术论文详情获取工具注册完成")

	// Create the streamable HTTP handler.
	log.Printf("[DEBUG] 正在创建HTTP处理程序...")
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		log.Printf("[DEBUG] HTTP请求处理: %s %s", req.Method, req.URL.Path)
		return server
	}, nil)
	log.Printf("[INFO] HTTP处理程序创建完成")

	handlerWithLogging := loggingHandler(handler)
	log.Printf("[DEBUG] 日志中间件已附加")

	url := "http://127.0.0.1:8080/"
	log.Printf("[INFO] 学术论文检索聚合MCP服务器正在监听: %s", url)
	log.Printf("[INFO] ")
	log.Printf("[INFO] 🎯 统一工具接口:")
	log.Printf("[INFO]   📚 searchScholarPapers - 聚合搜索学术论文")
	log.Printf("[INFO]     ✨ 自动调用多个数据源 (arXiv, Semantic Scholar, Crossref等)")
	log.Printf("[INFO]     ✨ 智能去重和结果合并")
	log.Printf("[INFO]     ✨ 支持高级过滤和排序")
	log.Printf("[INFO]   📄 getScholarPaper - 获取论文详情")
	log.Printf("[INFO]     ✨ 支持多种标识符 (DOI, arXiv ID, PubMed ID等)")
	log.Printf("[INFO]     ✨ 智能选择最佳数据源")
	log.Printf("[INFO] ")
	log.Printf("[INFO] 🔧 支持的数据源:")
	log.Printf("[INFO]   📚 arXiv - 物理学、数学、计算机科学预印本")
	log.Printf("[INFO]   🧠 Semantic Scholar - AI驱动的学术搜索")
	log.Printf("[INFO]   🔗 Crossref - DOI注册机构，广泛的期刊覆盖")
	log.Printf("[INFO]   📊 Scopus - Elsevier学术数据库 (需要API密钥)")
	log.Printf("[INFO]   🌌 ADSABS - 天体物理学文献 (需要API密钥)")
	log.Printf("[INFO]   📄 Sci-Hub - 论文PDF获取")
	log.Printf("[INFO] ")
	log.Printf("[INFO] 🔑 环境变量配置 (可选):")
	log.Printf("[INFO]   - SCOPUS_API_KEY: Scopus API密钥")
	log.Printf("[INFO]   - ADSABS_API_KEY: ADSABS API密钥")
	log.Printf("[INFO]   - CROSSREF_EMAIL: Crossref礼貌邮箱")
	log.Printf("[INFO] ")
	log.Printf("[INFO] 💡 使用示例:")
	log.Printf("[INFO]   基础搜索: {\"query\": \"machine learning\"}")
	log.Printf("[INFO]   高级搜索: {\"query\": \"deep learning\", \"author\": \"Hinton\", \"year\": \"2020-2023\", \"min_citations\": 100}")
	log.Printf("[INFO]   DOI查询: {\"identifier\": \"10.1038/nature12373\"}")

	// Start the HTTP server with logging handler.
	log.Printf("[INFO] ========== 服务器启动完成，等待连接 ==========")
	if err := http.ListenAndServe(":8080", handlerWithLogging); err != nil {
		log.Printf("[FATAL] 服务器启动失败: %v", err)
		log.Fatalf("Server failed: %v", err)
	}
}
