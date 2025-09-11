package aggregator

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Seelly/scholar_mcp_server/arxiv"
	"github.com/Seelly/scholar_mcp_server/semanticscholar"
	"github.com/Seelly/scholar_mcp_server/crossref"
	"github.com/Seelly/scholar_mcp_server/scopus"
	"github.com/Seelly/scholar_mcp_server/adsabs"
	"github.com/Seelly/scholar_mcp_server/scihub"
	"github.com/Seelly/scholar_mcp_server/common"
)

// ScholarAggregator 学术论文聚合器
type ScholarAggregator struct {
	config *common.Config
}

// NewScholarAggregator 创建新的聚合器
func NewScholarAggregator(config *common.Config) *ScholarAggregator {
	if config == nil {
		config = &common.Config{
			RequestTimeout:        30,
			MaxRetries:           3,
			EnableCache:          false,
			DefaultLimit:         10,
			MaxLimit:             100,
			DefaultSort:          "relevance",
			EnableSemanticScholar: true,
			EnableCrossref:       true,
			EnableScopus:         false, // 需要API密钥
			EnableAdsabs:         false, // 需要API密钥
			EnableSciHub:         true,
			EnableArxiv:          true,
		}
	}
	return &ScholarAggregator{config: config}
}

// SearchResult 聚合搜索结果
type SearchResult struct {
	Query          string               `json:"query"`
	TotalSources   int                  `json:"total_sources"`
	ActiveSources  int                  `json:"active_sources"`
	TotalResults   int                  `json:"total_results"`
	SearchTime     time.Duration        `json:"search_time"`
	Papers         []common.UnifiedPaper `json:"papers"`
	SourceResults  map[string]interface{} `json:"source_results"`
	SourceStatus   []common.ServiceStatus `json:"source_status"`
	Suggestions    []common.SearchHint  `json:"suggestions"`
}

// SearchPapers 聚合搜索论文
func (a *ScholarAggregator) SearchPapers(ctx context.Context, params common.SearchParams) (*SearchResult, error) {
	startTime := time.Now()
	
	// 验证参数
	if validation := common.ValidateSearchParams(params); !validation.Valid {
		return nil, fmt.Errorf("参数验证失败: %v", validation.Errors)
	}

	// 标准化参数
	common.NormalizeSearchParams(&params)

	log.Printf("[INFO] 开始聚合搜索: 关键词='%s', 限制=%d", params.Query, params.Limit)

	// 创建并发搜索通道
	type sourceResult struct {
		source  string
		papers  []common.UnifiedPaper
		total   int
		err     error
		elapsed time.Duration
	}

	resultChan := make(chan sourceResult, 6)
	var wg sync.WaitGroup
	totalSources := 0
	activeSources := 0

	// 并发搜索各个数据源
	if a.config.EnableArxiv {
		totalSources++
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			papers, total, err := a.searchArxiv(ctx, params)
			resultChan <- sourceResult{
				source:  "arxiv",
				papers:  papers,
				total:   total,
				err:     err,
				elapsed: time.Since(start),
			}
		}()
	}

	if a.config.EnableSemanticScholar {
		totalSources++
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			papers, total, err := a.searchSemanticScholar(ctx, params)
			resultChan <- sourceResult{
				source:  "semantic_scholar",
				papers:  papers,
				total:   total,
				err:     err,
				elapsed: time.Since(start),
			}
		}()
	}

	if a.config.EnableCrossref {
		totalSources++
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			papers, total, err := a.searchCrossref(ctx, params)
			resultChan <- sourceResult{
				source:  "crossref",
				papers:  papers,
				total:   total,
				err:     err,
				elapsed: time.Since(start),
			}
		}()
	}

	if a.config.EnableScopus && a.config.ScopusAPIKey != "" {
		totalSources++
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			papers, total, err := a.searchScopus(ctx, params)
			resultChan <- sourceResult{
				source:  "scopus",
				papers:  papers,
				total:   total,
				err:     err,
				elapsed: time.Since(start),
			}
		}()
	}

	if a.config.EnableAdsabs && a.config.AdsabsAPIKey != "" {
		totalSources++
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			papers, total, err := a.searchAdsabs(ctx, params)
			resultChan <- sourceResult{
				source:  "adsabs",
				papers:  papers,
				total:   total,
				err:     err,
				elapsed: time.Since(start),
			}
		}()
	}

	if a.config.EnableSciHub {
		totalSources++
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			papers, total, err := a.searchSciHub(ctx, params)
			resultChan <- sourceResult{
				source:  "scihub",
				papers:  papers,
				total:   total,
				err:     err,
				elapsed: time.Since(start),
			}
		}()
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
	a.sortPapers(mergedPapers, params.SortBy, params.SortOrder)

	// 应用分页
	if params.Offset >= len(mergedPapers) {
		mergedPapers = []common.UnifiedPaper{}
	} else {
		end := params.Offset + params.Limit
		if end > len(mergedPapers) {
			end = len(mergedPapers)
		}
		mergedPapers = mergedPapers[params.Offset:end]
	}

	// 生成搜索建议
	suggestions := common.GenerateSearchSuggestions(params.Query)

	searchTime := time.Since(startTime)
	log.Printf("[INFO] 聚合搜索完成: %d个数据源, %d个活跃, %d篇论文, 耗时%v", 
		totalSources, activeSources, len(mergedPapers), searchTime)

	return &SearchResult{
		Query:         params.Query,
		TotalSources:  totalSources,
		ActiveSources: activeSources,
		TotalResults:  totalResults,
		SearchTime:    searchTime,
		Papers:        mergedPapers,
		SourceResults: sourceResults,
		SourceStatus:  sourceStatus,
		Suggestions:   suggestions,
	}, nil
}

// GetPaper 根据标识符获取论文详情
func (a *ScholarAggregator) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	log.Printf("[INFO] 开始获取论文详情: %s", identifier)

	// 根据标识符类型选择最佳数据源
	var paper *common.UnifiedPaper
	var err error

	// 如果是DOI，优先使用Crossref
	if common.IsDOI(identifier) {
		if a.config.EnableCrossref {
			if p, e := a.getPaperFromCrossref(ctx, identifier); e == nil {
				paper = p
			}
		}
	}

	// 如果是arXiv ID，优先使用arXiv
	if paper == nil && common.IsArxivID(identifier) {
		if a.config.EnableArxiv {
			if p, e := a.getPaperFromArxiv(ctx, identifier); e == nil {
				paper = p
			}
		}
	}

	// 如果是PubMed ID，尝试Semantic Scholar
	if paper == nil && common.IsPubMedID(identifier) {
		if a.config.EnableSemanticScholar {
			if p, e := a.getPaperFromSemanticScholar(ctx, identifier); e == nil {
				paper = p
			}
		}
	}

	// 如果仍未找到，尝试所有数据源
	if paper == nil {
		paper, err = a.searchAllSourcesForPaper(ctx, identifier)
	}

	if paper == nil {
		return nil, fmt.Errorf("未找到标识符为 %s 的论文", identifier)
	}

	log.Printf("[INFO] 成功获取论文详情: %s", paper.Title)
	return paper, err
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
func (a *ScholarAggregator) sortPapers(papers []common.UnifiedPaper, sortBy, sortOrder string) {
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
			// 简单的相关性评分：引用数 + 开放获取奖励
			scoreI := papers[i].CitationCount
			scoreJ := papers[j].CitationCount
			if papers[i].IsOpenAccess {
				scoreI += 10
			}
			if papers[j].IsOpenAccess {
				scoreJ += 10
			}
			less = scoreI < scoreJ
		}
		
		if sortOrder == "asc" {
			return less
		}
		return !less
	})
}
