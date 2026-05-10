package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"

	sourcehandlers "github.com/Seelly/scholar_mcp_server/sources"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ScholarSearchParam 学术论文搜索参数
type ScholarSearchParam = sourcehandlers.ScholarSearchParam

// ScholarGetParam 获取论文详情参数
type ScholarGetParam = sourcehandlers.ScholarGetParam

// SciHubFetchParam 从 Sci-Hub 获取论文参数
type SciHubFetchParam = sourcehandlers.SciHubFetchParam

// SciHubPDFToMarkdownParam PDF 转 Markdown 参数
type SciHubPDFToMarkdownParam = sourcehandlers.SciHubPDFToMarkdownParam

// 统一的学术论文搜索功能
func searchScholarPapers(ctx context.Context, req *mcp.CallToolRequest, params *ScholarSearchParam) (*mcp.CallToolResult, any, error) {
	return sourcehandlers.NewUnifiedMCPHandler().SearchScholarPapers(ctx, req, params)
}

// 获取学术论文详情功能
func getScholarPaper(ctx context.Context, req *mcp.CallToolRequest, params *ScholarGetParam) (*mcp.CallToolResult, any, error) {
	return sourcehandlers.NewUnifiedMCPHandler().GetScholarPaper(ctx, req, params)
}

// 从 Sci-Hub 获取论文 PDF 链接
func fetchFromSciHub(ctx context.Context, req *mcp.CallToolRequest, params *SciHubFetchParam) (*mcp.CallToolResult, any, error) {
	return sourcehandlers.NewUnifiedMCPHandler().FetchFromSciHub(ctx, req, params)
}

// 从 Sci-Hub 下载 PDF 并转换为 Markdown
func sciHubPDFToMarkdown(ctx context.Context, req *mcp.CallToolRequest, params *SciHubPDFToMarkdownParam) (*mcp.CallToolResult, any, error) {
	return sourcehandlers.NewUnifiedMCPHandler().SciHubPDFToMarkdown(ctx, req, params)
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
		Description: "Search academic papers from multiple sources simultaneously (arXiv, Semantic Scholar, Crossref, OpenAlex, Scopus, ADSABS, optional Google Scholar via SerpAPI, Sci-Hub). Automatically aggregates, deduplicates, and ranks results. Supports advanced filtering by author, journal, year, citations, and open access status. Returns unified paper metadata with source attribution.",
	}, searchScholarPapers)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "getScholarPaper",
		Description: "Get detailed information of a specific academic paper by any identifier (DOI, arXiv ID, PubMed ID, Semantic Scholar ID, etc.). Automatically determines the best data source based on identifier type and retrieves comprehensive metadata including citations, abstracts, and external links.",
	}, getScholarPaper)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "fetchFromSciHub",
		Description: "After identifying the final list of papers to retrieve, use this tool to fetch the PDF download link for a paper from Sci-Hub by its DOI. Returns the PDF URL and Sci-Hub page link. If successful, you can then call sciHubPDFToMarkdown to extract the full text.",
	}, fetchFromSciHub)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sciHubPDFToMarkdown",
		Description: "Download the PDF of a paper from Sci-Hub by DOI and convert it to Markdown plain text. Use this after fetchFromSciHub confirms the paper is available. Returns the full paper content as Markdown, suitable for reading, summarizing, or further analysis.",
	}, sciHubPDFToMarkdown)

	return server
}

func runStdioServer() {
	// Redirect log output to stderr so it doesn't interfere with MCP protocol on stdout.
	log.SetOutput(os.Stderr)

	log.Printf("[INFO] ========== 学术论文检索聚合MCP服务器启动 (stdio) ==========")
	server := createServer()
	log.Printf("[INFO] 统一学术论文搜索工具已注册 (searchScholarPapers, getScholarPaper, fetchFromSciHub, sciHubPDFToMarkdown)")

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
	log.Printf("[INFO] 统一工具接口: searchScholarPapers, getScholarPaper, fetchFromSciHub, sciHubPDFToMarkdown")
	log.Printf("[INFO] 支持的数据源: arXiv, Semantic Scholar, Crossref, OpenAlex, Scopus, ADSABS, Google Scholar(SerpAPI), Sci-Hub")
	log.Printf("[INFO] ========== 服务器启动完成，等待连接 ==========")
	if err := http.ListenAndServe(":"+port, rootHandler); err != nil {
		log.Fatalf("[FATAL] 服务器启动失败: %v", err)
	}
}
