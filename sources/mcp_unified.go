package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Seelly/scholar_mcp_server/common"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UnifiedMCPHandler 统一MCP处理器
type UnifiedMCPHandler struct {
	aggregator *ScholarAggregator
}

// NewUnifiedMCPHandler 创建统一MCP处理器
func NewUnifiedMCPHandler() *UnifiedMCPHandler {
	return &UnifiedMCPHandler{
		aggregator: NewScholarAggregator(),
	}
}

// NewUnifiedMCPHandlerWithManager 使用指定的源管理器创建统一MCP处理器
func NewUnifiedMCPHandlerWithManager(manager *SourceManager) *UnifiedMCPHandler {
	return &UnifiedMCPHandler{
		aggregator: NewScholarAggregatorWithManager(manager),
	}
}

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

	// 格式化输出
	resultText := h.formatAggregatedSearchResult(result, params)

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] ========== 聚合搜索完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// GetScholarPaper 获取学术论文详情MCP工具
func (h *UnifiedMCPHandler) GetScholarPaper(ctx context.Context, req *mcp.CallToolRequest, params *ScholarGetParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始获取学术论文详情 ==========")
	log.Printf("[DEBUG] 接收到的参数: %+v", params)

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

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(paper)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] ========== 学术论文详情获取完成 ==========")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// SearchSourcePapers 单个数据源搜索MCP工具
func (h *UnifiedMCPHandler) SearchSourcePapers(ctx context.Context, req *mcp.CallToolRequest, params *SourceSearchParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始单个数据源搜索 ==========")
	log.Printf("[DEBUG] 接收到的参数: %+v", params)

	// 验证参数
	if strings.TrimSpace(params.Source) == "" {
		return nil, nil, fmt.Errorf("数据源名称不能为空")
	}
	if strings.TrimSpace(params.Query) == "" {
		return nil, nil, fmt.Errorf("搜索关键词不能为空")
	}

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

	log.Printf("[INFO] 开始单个数据源搜索: 源='%s', 关键词='%s'", params.Source, params.Query)

	// 执行搜索
	startTime := time.Now()
	papers, total, err := source.SearchPapers(ctx, searchParams)
	searchTime := time.Since(startTime)

	if err != nil {
		log.Printf("[ERROR] %s搜索失败: %v", params.Source, err)
		return nil, nil, fmt.Errorf("%s搜索失败: %w", params.Source, err)
	}

	// 格式化输出
	resultText := h.formatSourceSearchResult(params.Source, papers, total, searchTime, params)

	// 创建结果结构
	result := map[string]interface{}{
		"source":      params.Source,
		"query":       params.Query,
		"total":       total,
		"count":       len(papers),
		"search_time": searchTime.Milliseconds(),
		"papers":      papers,
	}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] %s搜索完成: %d篇论文, 耗时%v", params.Source, len(papers), searchTime)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
		},
	}, nil, nil
}

// GetSourcePaper 从单个数据源获取论文详情MCP工具
func (h *UnifiedMCPHandler) GetSourcePaper(ctx context.Context, req *mcp.CallToolRequest, params *SourceGetParam) (*mcp.CallToolResult, any, error) {
	log.Printf("[DEBUG] ========== 开始从单个数据源获取论文详情 ==========")
	log.Printf("[DEBUG] 接收到的参数: %+v", params)

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

	// 将结果转换为JSON
	jsonResult, err := json.Marshal(paper)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] 从%s成功获取论文详情: %s", params.Source, paper.Title)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
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

	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Printf("[ERROR] JSON序列化失败: %v", err)
	}

	log.Printf("[INFO] 数据源列表获取完成: 总计%d个, 已启用%d个", len(allSources), len(enabledSources))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultBuilder.String()},
		},
		Meta: map[string]interface{}{
			"structured_data": string(jsonResult),
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

	return resultBuilder.String()
}

func (h *UnifiedMCPHandler) formatPaperDetail(paper *common.UnifiedPaper) string {
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

	return resultBuilder.String()
}

func (h *UnifiedMCPHandler) formatSourceSearchResult(source string, papers []common.UnifiedPaper, total int, searchTime time.Duration, params *SourceSearchParam) string {
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("📚 %s 搜索结果 (关键词: '%s')\n", source, params.Query))
	resultBuilder.WriteString(fmt.Sprintf("搜索耗时: %v\n", searchTime))
	resultBuilder.WriteString(fmt.Sprintf("显示第%d-%d条结果，共找到约%d篇论文\n\n", params.Offset+1, params.Offset+len(papers), total))

	for i, paper := range papers {
		resultBuilder.WriteString(fmt.Sprintf("📄 %d. %s\n", i+1, paper.Title))
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
			if len(abstract) > 200 {
				abstract = abstract[:200] + "..."
			}
			resultBuilder.WriteString(fmt.Sprintf("   摘要: %s\n", abstract))
		}
		if paper.URL != "" {
			resultBuilder.WriteString(fmt.Sprintf("   链接: %s\n", paper.URL))
		}
		resultBuilder.WriteString("\n")
	}

	return resultBuilder.String()
}

func (h *UnifiedMCPHandler) formatSourcePaperDetail(source string, paper *common.UnifiedPaper) string {
	var resultBuilder strings.Builder
	resultBuilder.WriteString(fmt.Sprintf("📄 %s 论文详情\n\n", source))
	resultBuilder.WriteString(fmt.Sprintf("📝 标题: %s\n\n", paper.Title))

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
	if paper.PublishedDate != "" {
		resultBuilder.WriteString(fmt.Sprintf("📅 发表日期: %s\n", paper.PublishedDate))
	}
	if paper.CitationCount > 0 {
		resultBuilder.WriteString(fmt.Sprintf("📊 引用数: %d\n", paper.CitationCount))
	}

	if paper.IsOpenAccess {
		resultBuilder.WriteString("📖 开放获取: 是\n")
	}

	if len(paper.Categories) > 0 {
		resultBuilder.WriteString(fmt.Sprintf("🏷️ 分类: %s\n", strings.Join(paper.Categories, ", ")))
	}

	if paper.Abstract != "" {
		resultBuilder.WriteString(fmt.Sprintf("\n📋 摘要:\n%s\n", paper.Abstract))
	}

	if paper.URL != "" {
		resultBuilder.WriteString(fmt.Sprintf("\n🌐 原始链接: %s\n", paper.URL))
	}
	if paper.PDFURL != "" {
		resultBuilder.WriteString(fmt.Sprintf("📄 PDF下载: %s\n", paper.PDFURL))
	}

	return resultBuilder.String()
}
