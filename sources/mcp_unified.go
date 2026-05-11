package sources

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Seelly/scholar_mcp_server/common"
	"github.com/Seelly/scholar_mcp_server/sources/scihub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UnifiedMCPHandler 统一MCP处理器
type UnifiedMCPHandler struct {
	aggregator *ScholarAggregator
	enricher   *SearchResultEnricher
}

// NewUnifiedMCPHandler 创建统一MCP处理器
func NewUnifiedMCPHandler() *UnifiedMCPHandler {
	return &UnifiedMCPHandler{
		aggregator: NewScholarAggregator(),
		enricher:   NewConfiguredSearchResultEnricher(),
	}
}

// NewUnifiedMCPHandlerWithManager 使用指定的源管理器创建统一MCP处理器
func NewUnifiedMCPHandlerWithManager(manager *SourceManager) *UnifiedMCPHandler {
	return &UnifiedMCPHandler{
		aggregator: NewScholarAggregatorWithManager(manager),
		enricher:   NewConfiguredSearchResultEnricher(),
	}
}

// ScholarSearchParam 学术论文搜索参数
type ScholarSearchParam struct {
	Query          string   `json:"query"`                      // 搜索关键词
	Author         string   `json:"author,omitempty"`           // 作者筛选
	Title          string   `json:"title,omitempty"`            // 标题筛选
	Abstract       string   `json:"abstract,omitempty"`         // 摘要筛选
	Journal        string   `json:"journal,omitempty"`          // 期刊筛选
	Publisher      string   `json:"publisher,omitempty"`        // 出版商/机构筛选
	Year           string   `json:"year,omitempty"`             // 年份筛选
	YearRange      string   `json:"year_range,omitempty"`       // 年份范围筛选
	Language       string   `json:"language,omitempty"`         // 语言筛选
	Type           string   `json:"type,omitempty"`             // 文档类型筛选
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

// SourceSearchParam 单个数据源搜索参数
type SourceSearchParam struct {
	Source string `json:"source"` // 数据源名称
	ScholarSearchParam
}

// SourceGetParam 单个数据源获取论文参数
type SourceGetParam struct {
	Source     string `json:"source"`     // 数据源名称
	Identifier string `json:"identifier"` // 论文标识符
}

// SearchScholarPapers 统一学术论文搜索MCP工具
func (h *UnifiedMCPHandler) SearchScholarPapers(ctx context.Context, req *mcp.CallToolRequest, params *ScholarSearchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始聚合搜索学术论文 ==========")
	log.Printf("[DEBUG] 接收到的原始参数: %+v", params)

	// 转换为通用搜索参数并标准化
	searchParams := common.SearchParams{
		Query:          params.Query,
		Author:         params.Author,
		Title:          params.Title,
		Abstract:       params.Abstract,
		Journal:        params.Journal,
		Publisher:      params.Publisher,
		Year:           params.Year,
		YearRange:      params.YearRange,
		Language:       params.Language,
		Type:           params.Type,
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

	// 如果指定了启用的数据源，需要临时调整聚合器
	var originalManager *SourceManager
	if len(params.EnabledSources) > 0 {
		originalManager = h.aggregator.sourceManager
		h.aggregator.sourceManager = h.createFilteredManager(params.EnabledSources)
		defer func() {
			h.aggregator.sourceManager = originalManager
		}()
	}

	// 执行聚合搜索
	result, err := h.aggregator.SearchPapers(ctx, searchParams)
	if err != nil {
		log.Printf("[ERROR] 聚合搜索失败: %v", err)
		return nil, nil, fmt.Errorf("搜索失败: %w", err)
	}
	h.enricher.EnrichSearchResult(ctx, result)

	// 格式化输出
	resultText := h.formatAggregatedSearchResult(result, params)

	log.Printf("[INFO] ========== 聚合搜索完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
		Meta: map[string]interface{}{
			"structured_data": marshalStructuredData(result),
		},
	}, nil, nil
}

// GetScholarPaper 获取学术论文详情MCP工具
func (h *UnifiedMCPHandler) GetScholarPaper(ctx context.Context, req *mcp.CallToolRequest, params *ScholarGetParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始获取学术论文详情 ==========")
	log.Printf(logReceivedParamsFmt, params)

	// 验证标识符参数
	if strings.TrimSpace(params.Identifier) == "" {
		log.Printf("[ERROR] 论文标识符为空，终止请求")
		return nil, nil, fmt.Errorf("论文标识符不能为空")
	}

	log.Printf("[INFO] 开始获取论文详情: 标识符='%s'", params.Identifier)

	// 获取论文详情
	paper, err := h.aggregator.GetPaper(ctx, params.Identifier)
	if err != nil {
		log.Printf("[ERROR] 获取论文详情失败: %v", err)
		return nil, nil, fmt.Errorf("获取论文详情失败: %w", err)
	}

	// 格式化输出
	resultText := h.formatPaperDetail(paper)

	log.Printf("[INFO] ========== 学术论文详情获取完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
		Meta: map[string]interface{}{
			"structured_data": marshalStructuredData(paper),
		},
	}, nil, nil
}

// SearchSourcePapers 单个数据源搜索MCP工具
func (h *UnifiedMCPHandler) SearchSourcePapers(ctx context.Context, req *mcp.CallToolRequest, params *SourceSearchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始单个数据源搜索 ==========")
	log.Printf(logReceivedParamsFmt, params)

	// 验证参数
	if strings.TrimSpace(params.Source) == "" {
		return nil, nil, fmt.Errorf("数据源名称不能为空")
	}

	// 获取指定的数据源
	source, exists := h.aggregator.sourceManager.GetSource(params.Source)
	if !exists {
		return nil, nil, fmt.Errorf("数据源 '%s' 不存在或未启用", params.Source)
	}

	// 转换为通用搜索参数
	searchParams := common.SearchParams{
		Query:          params.Query,
		Author:         params.Author,
		Title:          params.Title,
		Abstract:       params.Abstract,
		Journal:        params.Journal,
		Publisher:      params.Publisher,
		Year:           params.Year,
		YearRange:      params.YearRange,
		Language:       params.Language,
		Type:           params.Type,
		Categories:     params.Categories,
		MinCitations:   params.MinCitations,
		OpenAccessOnly: params.OpenAccessOnly,
		Offset:         params.Offset,
		Limit:          params.Limit,
		SortBy:         params.SortBy,
		SortOrder:      params.SortOrder,
	}
	common.NormalizeSearchParams(&searchParams)
	if searchParams.Limit > 100 {
		searchParams.Limit = 100
	}
	if !common.HasSearchTerms(searchParams) {
		return nil, nil, fmt.Errorf("搜索条件不能为空，至少填写一个搜索词或筛选字段")
	}

	params.Query = searchParams.Query
	params.Offset = searchParams.Offset
	params.Limit = searchParams.Limit

	log.Printf("[INFO] 开始单个数据源搜索: 源='%s', 关键词='%s'", params.Source, searchParams.Query)

	// 执行搜索
	startTime := time.Now()
	papers, total, err := source.SearchPapers(ctx, searchParams)
	searchTime := time.Since(startTime)

	if err != nil {
		log.Printf("[ERROR] %s搜索失败: %v", params.Source, err)
		return nil, nil, fmt.Errorf("%s搜索失败: %w", params.Source, err)
	}
	h.enricher.EnrichPapers(ctx, papers)

	// 格式化输出
	resultText := h.formatSourceSearchResult(params.Source, papers, total, searchTime, params)

	// 创建结果结构
	result := map[string]interface{}{
		"source":      params.Source,
		"query":       searchParams.Query,
		"total":       total,
		"count":       len(papers),
		"search_time": searchTime.Milliseconds(),
		"papers":      papers,
	}

	log.Printf("[INFO] %s搜索完成: %d篇论文, 耗时%v", params.Source, len(papers), searchTime)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
		Meta: map[string]interface{}{
			"structured_data": marshalStructuredData(result),
		},
	}, nil, nil
}

// GetSourcePaper 从单个数据源获取论文详情MCP工具
func (h *UnifiedMCPHandler) GetSourcePaper(ctx context.Context, req *mcp.CallToolRequest, params *SourceGetParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始从单个数据源获取论文详情 ==========")
	log.Printf(logReceivedParamsFmt, params)

	// 验证参数
	if strings.TrimSpace(params.Source) == "" {
		return nil, nil, fmt.Errorf("数据源名称不能为空")
	}
	if strings.TrimSpace(params.Identifier) == "" {
		return nil, nil, fmt.Errorf("论文标识符不能为空")
	}

	// 获取指定的数据源
	source, exists := h.aggregator.sourceManager.GetSource(params.Source)
	if !exists {
		return nil, nil, fmt.Errorf("数据源 '%s' 不存在或未启用", params.Source)
	}

	log.Printf("[INFO] 开始从%s获取论文详情: 标识符='%s'", params.Source, params.Identifier)

	// 获取论文详情
	paper, err := source.GetPaper(ctx, params.Identifier)
	if err != nil {
		log.Printf("[ERROR] 从%s获取论文详情失败: %v", params.Source, err)
		return nil, nil, fmt.Errorf("从%s获取论文详情失败: %w", params.Source, err)
	}

	// 格式化输出
	resultText := h.formatSourcePaperDetail(params.Source, paper)

	log.Printf("[INFO] 从%s成功获取论文详情: %s", params.Source, paper.Title)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
		Meta: map[string]interface{}{
			"structured_data": marshalStructuredData(paper),
		},
	}, nil, nil
}

// ListAvailableSources 列出可用数据源MCP工具
func (h *UnifiedMCPHandler) ListAvailableSources(ctx context.Context, req *mcp.CallToolRequest, params *map[string]interface{}) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始列出可用数据源 ==========")

	allSources := h.aggregator.sourceManager.GetAllSources()
	enabledSources := h.aggregator.sourceManager.GetEnabledSources()

	var resultBuilder strings.Builder
	resultBuilder.WriteString("📚 学术论文数据源列表\n\n")

	resultBuilder.WriteString("✅ 已启用的数据源:\n")
	if len(enabledSources) == 0 {
		resultBuilder.WriteString("   无已启用的数据源\n")
	} else {
		for name, source := range enabledSources {
			capabilities := source.GetCapabilities()
			isAvailable := source.IsAvailable(ctx)

			status := "🔴 不可用"
			if isAvailable {
				status = "🟢 可用"
			}

			resultBuilder.WriteString(fmt.Sprintf("   • %s %s\n", name, status))
			resultBuilder.WriteString(fmt.Sprintf("     - 支持全文搜索: %v\n", capabilities.SupportsFullText))
			resultBuilder.WriteString(fmt.Sprintf("     - 支持作者搜索: %v\n", capabilities.SupportsAuthorSearch))
			resultBuilder.WriteString(fmt.Sprintf("     - 支持引用信息: %v\n", capabilities.SupportsCitation))
			resultBuilder.WriteString(fmt.Sprintf("     - 支持PDF获取: %v\n", capabilities.SupportsPDF))
			resultBuilder.WriteString(fmt.Sprintf("     - 需要API密钥: %v\n", capabilities.RequiresAPIKey))
			resultBuilder.WriteString(fmt.Sprintf("     - 每页最大结果: %d\n", capabilities.MaxResultsPerPage))
			resultBuilder.WriteString("\n")
		}
	}

	resultBuilder.WriteString("📋 所有已注册的数据源:\n")
	for name := range allSources {
		enabled := "❌ 未启用"
		if _, exists := enabledSources[name]; exists {
			enabled = "✅ 已启用"
		}
		resultBuilder.WriteString(fmt.Sprintf("   • %s %s\n", name, enabled))
	}

	// 创建结果数据
	sourceInfo := make(map[string]interface{})
	for name, source := range allSources {
		capabilities := source.GetCapabilities()
		isEnabled := false
		isAvailable := false

		if _, exists := enabledSources[name]; exists {
			isEnabled = true
			isAvailable = source.IsAvailable(ctx)
		}

		sourceInfo[name] = map[string]interface{}{
			"enabled":      isEnabled,
			"available":    isAvailable,
			"capabilities": capabilities,
		}
	}

	result := map[string]interface{}{
		"total_sources":   len(allSources),
		"enabled_sources": len(enabledSources),
		"sources":         sourceInfo,
	}

	log.Printf("[INFO] 数据源列表获取完成: 总计%d个, 已启用%d个", len(allSources), len(enabledSources))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": marshalStructuredData(result),
		},
	}, nil, nil
}

// createFilteredManager 创建过滤后的源管理器
func (h *UnifiedMCPHandler) createFilteredManager(enabledSources []string) *SourceManager {
	filteredManager := NewSourceManager()

	for _, sourceName := range enabledSources {
		if source, exists := h.aggregator.sourceManager.GetSource(sourceName); exists {
			// 获取原始配置
			allSources := h.aggregator.sourceManager.GetAllSources()
			if _, ok := allSources[sourceName]; ok {
				// 创建启用的配置
				config := SourceConfig{Enabled: true}
				filteredManager.RegisterSource(sourceName, source, config)
			}
		}
	}

	return filteredManager
}

// 格式化方法...
func (h *UnifiedMCPHandler) formatAggregatedSearchResult(result *AggregatedSearchResult, params *ScholarSearchParam) string {
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("🔬 学术论文聚合搜索结果 (关键词: '%s')\n", params.Query))
	resultBuilder.WriteString(fmt.Sprintf("数据源: %d个总计, %d个活跃\n", result.TotalSources, result.ActiveSources))
	resultBuilder.WriteString(fmt.Sprintf("搜索耗时: %v\n", result.SearchTime))
	resultBuilder.WriteString(fmt.Sprintf("显示第%d-%d条结果，共找到约%d篇论文\n\n", params.Offset+1, params.Offset+len(result.Papers), result.TotalResults))

	for i, paper := range result.Papers {
		resultBuilder.WriteString(formatPaperSearchEntry(i+1, paper, true, 300))
	}

	resultBuilder.WriteString(formatSourceStatusSection(result.SourceStatus))
	resultBuilder.WriteString(formatSuggestionsSection(result.Suggestions))

	return resultBuilder.String()
}

func (h *UnifiedMCPHandler) formatPaperDetail(paper *common.UnifiedPaper) string {
	var resultBuilder strings.Builder
	resultBuilder.WriteString("📄 学术论文详情\n\n")
	resultBuilder.WriteString(fmt.Sprintf("📝 标题: %s\n\n", paper.Title))
	resultBuilder.WriteString(formatPaperDetailBody(paper, true))

	return resultBuilder.String()
}

func (h *UnifiedMCPHandler) formatSourceSearchResult(source string, papers []common.UnifiedPaper, total int, searchTime time.Duration, params *SourceSearchParam) string {
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("📚 %s 搜索结果 (关键词: '%s')\n", source, params.Query))
	resultBuilder.WriteString(fmt.Sprintf("搜索耗时: %v\n", searchTime))
	resultBuilder.WriteString(fmt.Sprintf("显示第%d-%d条结果，共找到约%d篇论文\n\n", params.Offset+1, params.Offset+len(papers), total))

	for i, paper := range papers {
		resultBuilder.WriteString(formatPaperSearchEntry(i+1, paper, false, 200))
	}

	return resultBuilder.String()
}

func (h *UnifiedMCPHandler) formatSourcePaperDetail(source string, paper *common.UnifiedPaper) string {
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("📄 %s 论文详情\n\n", source))
	resultBuilder.WriteString(fmt.Sprintf("📝 标题: %s\n\n", paper.Title))
	resultBuilder.WriteString(formatPaperDetailBody(paper, false))

	return resultBuilder.String()
}

// SciHubFetchParam 从 Sci-Hub 获取论文参数
type SciHubFetchParam struct {
	DOI string `json:"doi"` // 论文 DOI
}

// SciHubPDFToMarkdownParam 从 Sci-Hub 下载 PDF 并转换为 Markdown 的参数
type SciHubPDFToMarkdownParam struct {
	DOI string `json:"doi"` // 论文 DOI
}

// FetchFromSciHub 通过 DOI 从 Sci-Hub 获取论文 PDF 链接
func (h *UnifiedMCPHandler) FetchFromSciHub(ctx context.Context, req *mcp.CallToolRequest, params *SciHubFetchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始从 Sci-Hub 获取论文 ==========")
	log.Printf(logReceivedParamsFmt, params)

	doi := strings.TrimSpace(params.DOI)
	if doi == "" {
		return nil, nil, fmt.Errorf("DOI 不能为空")
	}

	client := scihub.NewSciHubClient()
	resp, err := client.GetPDFURL(ctx, doi)
	if err != nil || !resp.Success {
		msg := "未能从 Sci-Hub 获取该论文"
		if err != nil {
			msg = err.Error()
		} else if resp != nil {
			msg = resp.Message
		}
		log.Printf("[WARN] Sci-Hub 获取失败: %s", msg)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("❌ 无法从 Sci-Hub 获取 DOI 为 %s 的论文。\n原因: %s\n\n建议检查 DOI 是否正确，或稍后重试。", doi, msg)},
			},
		}, nil, nil
	}

	var sb strings.Builder
	sb.WriteString("✅ 已从 Sci-Hub 找到论文 PDF\n\n")
	sb.WriteString(fmt.Sprintf("📄 DOI: %s\n", doi))
	sb.WriteString(fmt.Sprintf("🔗 PDF 链接: %s\n", resp.PDFURL))
	sb.WriteString(fmt.Sprintf("🌐 Sci-Hub 页面: %s\n", resp.SciHubURL))
	if resp.FileSize != "" {
		sb.WriteString(fmt.Sprintf("📦 文件大小: %s\n", resp.FileSize))
	}
	sb.WriteString("\n💡 提示: 可使用 sciHubPDFToMarkdown 工具将该 PDF 转换为 Markdown 全文。")

	log.Printf("[INFO] Sci-Hub 获取成功: DOI=%s, PDF=%s", doi, resp.PDFURL)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: sb.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": marshalStructuredData(resp),
		},
	}, nil, nil
}

// SciHubPDFToMarkdown 下载 Sci-Hub PDF 并转换为 Markdown 全文
func (h *UnifiedMCPHandler) SciHubPDFToMarkdown(ctx context.Context, req *mcp.CallToolRequest, params *SciHubPDFToMarkdownParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始 Sci-Hub PDF 转 Markdown ==========")
	log.Printf(logReceivedParamsFmt, params)

	doi := strings.TrimSpace(params.DOI)
	if doi == "" {
		return nil, nil, fmt.Errorf("DOI 不能为空")
	}

	client := scihub.NewSciHubClient()
	markdown, err := client.DownloadAndConvertPDF(ctx, doi)
	if err != nil {
		log.Printf("[ERROR] PDF 转换失败: %v", err)
		return nil, nil, fmt.Errorf("PDF 转换失败: %w", err)
	}

	if strings.TrimSpace(markdown) == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("⚠️ 已下载 DOI 为 %s 的 PDF，但未能提取到文本内容（可能为扫描版 PDF）。", doi)},
			},
		}, nil, nil
	}

	log.Printf("[INFO] PDF 转 Markdown 完成: DOI=%s, 字符数=%d", doi, len(markdown))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: markdown},
		},
		Meta: map[string]interface{}{
			"structured_data": marshalStructuredData(map[string]interface{}{
				"doi":            doi,
				"char_count":     len(markdown),
				"content_format": "markdown",
			}),
		},
	}, nil, nil
}
