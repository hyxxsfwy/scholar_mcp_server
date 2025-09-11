package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"log"
	"mcp/arxiv"
	"net/http"
	"strings"
)

// arXiv搜索功能
func searchArxivPapers(ctx context.Context, req *mcp.CallToolRequest, params *arxiv.ArxivSearchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始搜索arXiv论文 ==========")
	log.Printf("[DEBUG] 接收到的原始参数: %+v", params)

	// 设置默认值
	if params.Start < 0 {
		params.Start = 0
	}
	if params.MaxResults <= 0 {
		params.MaxResults = 10
	}
	if params.MaxResults > 100 {
		params.MaxResults = 100
	}

	log.Printf("[DEBUG] 参数验证后的最终值: Query='%s', Start=%d, MaxResults=%d", params.Query, params.Start, params.MaxResults)

	// 验证查询参数
	if strings.TrimSpace(params.Query) == "" {
		log.Printf("[ERROR] 搜索关键词为空，终止请求")
		return nil, nil, fmt.Errorf("搜索关键词不能为空")
	}

	log.Printf("[INFO] 开始搜索arXiv: 关键词='%s', Start=%d, MaxResults=%d", params.Query, params.Start, params.MaxResults)

	// 创建arXiv客户端并搜索
	client := arxiv.NewArxivClient()
	result, err := client.SearchPapers(ctx, params.Query, params.Start, params.MaxResults)
	if err != nil {
		log.Printf("[ERROR] arXiv搜索失败: %v", err)
		return nil, nil, fmt.Errorf("搜索失败: %w", err)
	}

	// 格式化输出
	log.Printf("[DEBUG] 开始格式化搜索结果文本输出...")
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("🔬 arXiv论文搜索结果 (关键词: '%s')\n", params.Query))
	resultBuilder.WriteString(fmt.Sprintf("显示第%d-%d条结果，共找到约%d篇论文\n\n", params.Start+1, params.Start+len(result.Papers), result.TotalCount))

	for i, paper := range result.Papers {
		resultBuilder.WriteString(fmt.Sprintf("📄 %d. %s\n", i+1, paper.Title))
		resultBuilder.WriteString(fmt.Sprintf("   ID: %s\n", paper.ID))
		resultBuilder.WriteString(fmt.Sprintf("   作者: %s\n", strings.Join(paper.Authors, ", ")))
		if paper.Published != "" {
			resultBuilder.WriteString(fmt.Sprintf("   发布日期: %s\n", paper.Published))
		}
		if len(paper.Categories) > 0 {
			resultBuilder.WriteString(fmt.Sprintf("   分类: %s\n", strings.Join(paper.Categories, ", ")))
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
	log.Printf("[DEBUG] 文本格式化完成，长度: %d 字符", resultBuilder.Len())

	// 将结果转换为JSON用于结构化数据
	log.Printf("[DEBUG] 开始JSON序列化搜索结果...")
	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	} else {
		log.Printf("[DEBUG] JSON序列化成功，长度: %d 字节", len(jsonResult))
	}

	log.Printf("[INFO] ========== arXiv搜索完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// 获取arXiv论文详情功能
func getArxivPaper(ctx context.Context, req *mcp.CallToolRequest, params *arxiv.ArxivGetParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始获取arXiv论文详情 ==========")
	log.Printf("[DEBUG] 接收到的参数: %+v", params)

	// 验证ID参数
	if strings.TrimSpace(params.ID) == "" {
		log.Printf("[ERROR] arXiv ID为空，终止请求")
		return nil, nil, fmt.Errorf("arXiv ID不能为空")
	}

	log.Printf("[INFO] 开始获取arXiv论文详情: ID='%s'", params.ID)

	// 创建arXiv客户端并获取论文
	client := arxiv.NewArxivClient()
	paper, err := client.GetPaper(ctx, params.ID)
	if err != nil {
		log.Printf("[ERROR] 获取arXiv论文详情失败: %v", err)
		return nil, nil, fmt.Errorf("获取论文详情失败: %w", err)
	}

	// 格式化输出
	log.Printf("[DEBUG] 开始格式化论文详情输出...")
	var resultBuilder strings.Builder
	resultBuilder.WriteString("📄 arXiv论文详情\n\n")
	resultBuilder.WriteString(fmt.Sprintf("🆔 ID: %s\n\n", paper.ID))
	resultBuilder.WriteString(fmt.Sprintf("📝 标题: %s\n\n", paper.Title))

	if len(paper.Authors) > 0 {
		resultBuilder.WriteString(fmt.Sprintf("👥 作者: %s\n\n", strings.Join(paper.Authors, ", ")))
	}

	if paper.Published != "" {
		resultBuilder.WriteString(fmt.Sprintf("📅 发布日期: %s\n", paper.Published))
	}

	if paper.Updated != "" {
		resultBuilder.WriteString(fmt.Sprintf("🔄 更新日期: %s\n", paper.Updated))
	}

	if len(paper.Categories) > 0 {
		resultBuilder.WriteString(fmt.Sprintf("🏷️ 分类: %s\n\n", strings.Join(paper.Categories, ", ")))
	}

	if paper.Abstract != "" {
		resultBuilder.WriteString(fmt.Sprintf("📋 摘要:\n%s\n\n", paper.Abstract))
	}

	if paper.JournalRef != "" {
		resultBuilder.WriteString(fmt.Sprintf("📖 期刊引用: %s\n", paper.JournalRef))
	}

	if paper.DOI != "" {
		resultBuilder.WriteString(fmt.Sprintf("🔗 DOI: %s\n", paper.DOI))
	}

	if paper.Comment != "" {
		resultBuilder.WriteString(fmt.Sprintf("💬 注释: %s\n", paper.Comment))
	}

	if paper.URL != "" {
		resultBuilder.WriteString(fmt.Sprintf("🌐 arXiv链接: %s\n", paper.URL))
	}

	if paper.PDFURL != "" {
		resultBuilder.WriteString(fmt.Sprintf("📄 PDF下载: %s\n", paper.PDFURL))
	}

	log.Printf("[DEBUG] 论文详情文本格式化完成，长度: %d 字符", resultBuilder.Len())

	// 将结果转换为JSON
	log.Printf("[DEBUG] 开始JSON序列化论文详情...")
	jsonResult, err := json.Marshal(paper)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	} else {
		log.Printf("[DEBUG] JSON序列化成功，长度: %d 字节", len(jsonResult))
	}

	log.Printf("[INFO] ========== arXiv论文详情获取完成 ==========")
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
	log.Printf("[INFO] ========== arXiv MCP服务器启动 ==========")
	log.Printf("[INFO] 正在初始化arXiv论文检索MCP服务器...")

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "arxiv-mcp-server",
		Version: "1.0.0",
	}, nil)
	log.Printf("[DEBUG] MCP服务器创建完成，名称: %s, 版本: %s", "arxiv-mcp-server", "1.0.0")

	// 添加arXiv论文搜索工具
	log.Printf("[DEBUG] 正在注册arXiv论文搜索工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "searchArxivPapers",
		Description: "Search academic papers from arXiv. Supports keywords, categories, authors, and advanced search queries. Returns paper titles, authors, abstracts, categories, and download links.",
	}, searchArxivPapers)
	log.Printf("[INFO] arXiv论文搜索工具注册完成")

	// 添加获取arXiv论文详情工具
	log.Printf("[DEBUG] 正在注册arXiv论文详情获取工具...")
	mcp.AddTool(server, &mcp.Tool{
		Name:        "getArxivPaper",
		Description: "Get detailed information of a specific paper by arXiv ID. Returns complete paper metadata including title, authors, abstract, categories, publication dates, and download links.",
	}, getArxivPaper)
	log.Printf("[INFO] arXiv论文详情获取工具注册完成")

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
	log.Printf("[INFO] arXiv MCP服务器正在监听: %s", url)
	log.Printf("[INFO] 可用的工具:")
	log.Printf("[INFO]   - searchArxivPapers: 搜索arXiv论文")
	log.Printf("[INFO]   - getArxivPaper: 获取特定arXiv论文的详细信息")
	log.Printf("[INFO] ")
	log.Printf("[INFO] 使用示例:")
	log.Printf("[INFO]   搜索: {\"query\": \"machine learning\", \"max_results\": 5}")
	log.Printf("[INFO]   获取: {\"id\": \"2301.07041\"}")

	// Start the HTTP server with logging handler.
	log.Printf("[INFO] ========== 服务器启动完成，等待连接 ==========")
	if err := http.ListenAndServe(":8080", handlerWithLogging); err != nil {
		log.Printf("[FATAL] 服务器启动失败: %v", err)
		log.Fatalf("Server failed: %v", err)
	}
}