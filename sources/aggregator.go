package sources

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Seelly/scholar_mcp_server/common"
)

// ScholarAggregator 学术论文聚合器
type ScholarAggregator struct {
	sourceManager *SourceManager
}

// NewScholarAggregator 创建新的聚合器
func NewScholarAggregator() *ScholarAggregator {
	return &ScholarAggregator{
		sourceManager: GetDefaultSourceManager(),
	}
}

// NewScholarAggregatorWithManager 使用指定的数据源管理器创建聚合器
func NewScholarAggregatorWithManager(manager *SourceManager) *ScholarAggregator {
	return &ScholarAggregator{
		sourceManager: manager,
	}
}

// AggregatedSearchResult 聚合搜索结果
type AggregatedSearchResult struct {
	Query         string                 `json:"query"`
	TotalSources  int                    `json:"total_sources"`
	ActiveSources int                    `json:"active_sources"`
	TotalResults  int                    `json:"total_results"`
	SearchTime    time.Duration          `json:"search_time"`
	Papers        []common.UnifiedPaper  `json:"papers"`
	SourceResults map[string]interface{} `json:"source_results"`
	SourceStatus  []common.ServiceStatus `json:"source_status"`
	Suggestions   []common.SearchHint    `json:"suggestions"`
	SearchPlan    *SemanticSearchPlan    `json:"search_plan,omitempty"`
}

// SearchPapers 聚合搜索论文
func (a *ScholarAggregator) SearchPapers(ctx context.Context, params common.SearchParams) (*AggregatedSearchResult, error) {
	startTime := time.Now()

	// 验证参数
	if validation := common.ValidateSearchParams(params); !validation.Valid {
		return nil, fmt.Errorf("参数验证失败: %v", validation.Errors)
	}

	// 标准化参数
	common.NormalizeSearchParams(&params)
	requestedOffset := params.Offset
	requestedLimit := params.Limit
	plan := buildSemanticSearchPlan(ctx, params)
	searchCtx := contextWithSemanticSearchPlan(ctx, plan)
	searchParams := applySemanticSearchPlan(params, plan)
	searchParams.Offset = 0
	searchParams.Limit = semanticCandidateLimit(requestedOffset, requestedLimit)

	log.Printf("[INFO] 开始聚合搜索: 关键词='%s', 限制=%d", searchParams.Query, searchParams.Limit)

	// 创建并发搜索通道
	type sourceResult struct {
		source  string
		papers  []common.UnifiedPaper
		total   int
		err     error
		elapsed time.Duration
	}

	resultChan := make(chan sourceResult, 10) // 增加缓冲区大小
	var wg sync.WaitGroup
	totalSources := 0
	activeSources := 0

	// 并发搜索各个数据源
	enabledSources := a.sourceManager.GetEnabledSources()
	totalSources = len(enabledSources)

	for sourceName, source := range enabledSources {
		wg.Add(1)
		go func(name string, src PaperSource) {
			defer wg.Done()
			start := time.Now()
			papers, total, err := src.SearchPapers(searchCtx, searchParams)
			resultChan <- sourceResult{
				source:  name,
				papers:  papers,
				total:   total,
				err:     err,
				elapsed: time.Since(start),
			}
		}(sourceName, source)
	}

	// 等待所有搜索完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	var allPapers []common.UnifiedPaper
	sourceResults := make(map[string]interface{})
	var sourceStatus []common.ServiceStatus
	totalResults := 0

	for result := range resultChan {
		if result.err != nil {
			log.Printf("[ERROR] %s搜索失败: %v", result.source, result.err)
			sourceStatus = append(sourceStatus, common.ServiceStatus{
				Source:       result.source,
				Available:    false,
				ResponseTime: result.elapsed.Milliseconds(),
				LastCheck:    time.Now().Format(time.RFC3339),
				Message:      result.err.Error(),
			})
		} else {
			activeSources++
			allPapers = append(allPapers, result.papers...)
			totalResults += result.total
			sourceResults[result.source] = map[string]interface{}{
				"papers": result.papers,
				"total":  result.total,
			}
			sourceStatus = append(sourceStatus, common.ServiceStatus{
				Source:       result.source,
				Available:    true,
				ResponseTime: result.elapsed.Milliseconds(),
				LastCheck:    time.Now().Format(time.RFC3339),
				Message:      fmt.Sprintf("找到%d篇论文", len(result.papers)),
			})
			log.Printf("[INFO] %s搜索完成: %d篇论文, 耗时%v", result.source, len(result.papers), result.elapsed)
		}
	}

	// 合并和去重
	mergedPapers := a.mergePapers(allPapers)

	// 排序
	a.sortPapers(mergedPapers, params.SortBy, params.SortOrder, plan)
	rerankTopSemanticWindow(ctx, mergedPapers, plan, semanticRerankWindow)

	// 应用分页
	mergedPapers = pageSemanticPapers(mergedPapers, requestedOffset, requestedLimit)

	// 生成搜索建议
	suggestions := append(semanticSearchHints(plan), common.GenerateSearchSuggestions(params.Query)...)

	searchTime := time.Since(startTime)
	log.Printf("[INFO] 聚合搜索完成: %d个数据源, %d个活跃, %d篇论文, 耗时%v",
		totalSources, activeSources, len(mergedPapers), searchTime)

	return &AggregatedSearchResult{
		Query:         searchParams.Query,
		TotalSources:  totalSources,
		ActiveSources: activeSources,
		TotalResults:  totalResults,
		SearchTime:    searchTime,
		Papers:        mergedPapers,
		SourceResults: sourceResults,
		SourceStatus:  sourceStatus,
		Suggestions:   suggestions,
		SearchPlan:    plan,
	}, nil
}

// GetPaper 根据标识符获取论文详情
func (a *ScholarAggregator) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	log.Printf("[INFO] 开始获取论文详情: %s", identifier)

	preferredSources := preferredSourcesForIdentifier(identifier)
	if paper, ok := a.tryGetPaperFromPreferredSources(ctx, identifier, preferredSources); ok {
		return paper, nil
	}
	if paper, ok := a.tryGetPaperFromRemainingSources(ctx, identifier, preferredSources); ok {
		return paper, nil
	}

	return nil, fmt.Errorf("未找到标识符为 %s 的论文", identifier)
}

func preferredSourcesForIdentifier(identifier string) []string {
	if common.IsDOI(identifier) {
		return []string{"crossref", "semantic_scholar", "openalex"}
	}
	if common.IsArxivID(identifier) {
		return []string{"arxiv", "semantic_scholar"}
	}
	if common.IsPubMedID(identifier) {
		return []string{"semantic_scholar"}
	}

	return nil
}

func (a *ScholarAggregator) tryGetPaperFromPreferredSources(ctx context.Context, identifier string, sourceNames []string) (*common.UnifiedPaper, bool) {
	for _, sourceName := range sourceNames {
		if paper, ok := a.tryGetPaperFromSource(ctx, identifier, sourceName); ok {
			return paper, true
		}
	}

	return nil, false
}

func (a *ScholarAggregator) tryGetPaperFromRemainingSources(ctx context.Context, identifier string, skippedSources []string) (*common.UnifiedPaper, bool) {
	skipped := sourceNameSet(skippedSources)
	for sourceName := range a.sourceManager.GetEnabledSources() {
		if skipped[sourceName] {
			continue
		}
		if paper, ok := a.tryGetPaperFromSource(ctx, identifier, sourceName); ok {
			return paper, true
		}
	}

	return nil, false
}

func (a *ScholarAggregator) tryGetPaperFromSource(ctx context.Context, identifier, sourceName string) (*common.UnifiedPaper, bool) {
	source, exists := a.sourceManager.GetSource(sourceName)
	if !exists {
		return nil, false
	}

	paper, err := source.GetPaper(ctx, identifier)
	if err != nil {
		return nil, false
	}

	log.Printf("[INFO] 从%s成功获取论文详情: %s", sourceName, paper.Title)
	return paper, true
}

func sourceNameSet(sourceNames []string) map[string]bool {
	set := make(map[string]bool, len(sourceNames))
	for _, sourceName := range sourceNames {
		set[sourceName] = true
	}

	return set
}

// mergePapers 合并和去重论文
func (a *ScholarAggregator) mergePapers(papers []common.UnifiedPaper) []common.UnifiedPaper {
	// 使用DOI和标题进行去重
	seen := make(map[string]bool)
	var merged []common.UnifiedPaper

	for _, paper := range papers {
		// 生成唯一键
		key := a.generatePaperKey(paper)
		if !seen[key] {
			seen[key] = true
			merged = append(merged, paper)
		}
	}

	log.Printf("[INFO] 去重前: %d篇论文, 去重后: %d篇论文", len(papers), len(merged))
	return merged
}

// generatePaperKey 生成论文的唯一键
func (a *ScholarAggregator) generatePaperKey(paper common.UnifiedPaper) string {
	// 优先使用DOI
	if paper.DOI != "" {
		return "doi:" + strings.ToLower(paper.DOI)
	}

	// 其次使用标题（标准化后）
	if paper.Title != "" {
		title := strings.ToLower(strings.TrimSpace(paper.Title))
		// 移除标点符号和多余空格
		title = strings.ReplaceAll(title, ".", "")
		title = strings.ReplaceAll(title, ",", "")
		title = strings.ReplaceAll(title, ":", "")
		title = strings.ReplaceAll(title, ";", "")
		title = strings.ReplaceAll(title, "?", "")
		title = strings.ReplaceAll(title, "!", "")
		words := strings.Fields(title)
		if len(words) > 5 {
			// 使用前5个单词作为键
			return "title:" + strings.Join(words[:5], " ")
		}
		return "title:" + title
	}

	// 最后使用ID
	return "id:" + paper.ID
}

// sortPapers 排序论文
func (a *ScholarAggregator) sortPapers(papers []common.UnifiedPaper, sortBy, sortOrder string, plan *SemanticSearchPlan) {
	sort.Slice(papers, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "citation_count":
			less = papers[i].CitationCount < papers[j].CitationCount
		case "published_date":
			less = papers[i].PublishedDate < papers[j].PublishedDate
		case "title":
			less = papers[i].Title < papers[j].Title
		default: // relevance
			scoreI := semanticPaperScore(papers[i], plan)
			scoreJ := semanticPaperScore(papers[j], plan)
			less = scoreI < scoreJ
		}

		if sortOrder == "asc" {
			return less
		}
		return !less
	})
}
