package sources

import (
	"context"

	"github.com/Seelly/scholar_mcp_server/sources/crossref"

	"github.com/Seelly/scholar_mcp_server/common"
)

// CrossrefSource Crossref数据源实现
type CrossrefSource struct {
	client *crossref.CrossrefClient
	config SourceConfig
}

// NewCrossrefSource 创建新的Crossref数据源
func NewCrossrefSource(config SourceConfig) *CrossrefSource {
	var client *crossref.CrossrefClient
	if config.APIKey != "" {
		client = crossref.NewCrossrefClientWithEmail(config.APIKey) // 这里APIKey用作邮箱
	} else {
		client = crossref.NewCrossrefClient()
	}

	return &CrossrefSource{
		client: client,
		config: config,
	}
}

// GetSourceName 获取数据源名称
func (c *CrossrefSource) GetSourceName() string {
	return "crossref"
}

// SearchPapers 搜索论文
func (c *CrossrefSource) SearchPapers(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	result, err := c.searchResult(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	papers := make([]common.UnifiedPaper, 0, len(result.Papers))
	for _, paper := range result.Papers {
		papers = append(papers, newCrossrefUnifiedPaper(paper))
	}

	return papers, result.TotalResults, nil
}

// GetPaper 获取特定论文
func (c *CrossrefSource) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	paper, err := c.client.GetPaper(ctx, identifier)
	if err != nil {
		return nil, err
	}

	unified := newCrossrefUnifiedPaper(*paper)
	return &unified, nil
}

func (c *CrossrefSource) searchResult(ctx context.Context, params common.SearchParams) (*crossref.SearchResult, error) {
	if hasCrossrefStructuredFilters(params) {
		return c.client.SearchPapersWithParams(ctx, buildCrossrefSearchParam(params))
	}

	return c.client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
}

func newCrossrefUnifiedPaper(paper crossref.Paper) common.UnifiedPaper {
	return common.UnifiedPaper{
		ID:            paper.DOI,
		DOI:           paper.DOI,
		Title:         firstCrossrefValue(paper.Title),
		Authors:       crossrefAuthorNames(paper.Author),
		Journal:       firstCrossrefValue(paper.ContainerTitle),
		Publisher:     paper.Publisher,
		Volume:        paper.Volume,
		Issue:         paper.Issue,
		Pages:         paper.Page,
		CitationCount: paper.IsReferencedByCount,
		URL:           paper.URL,
		Type:          paper.Type,
		Source:        "crossref",
		ExternalIDs: map[string]string{
			"doi": paper.DOI,
		},
		Metadata: map[string]interface{}{
			"references_count": paper.ReferencesCount,
			"license":          paper.License,
			"funder":           paper.Funder,
			"language":         paper.Language,
			"subject":          paper.Subject,
		},
	}
}

func crossrefAuthorNames(authors []crossref.Author) []string {
	names := make([]string, 0, len(authors))
	for _, author := range authors {
		name := crossrefAuthorName(author)
		if name != "" {
			names = append(names, name)
		}
	}

	return names
}

func crossrefAuthorName(author crossref.Author) string {
	if author.Given != "" && author.Family != "" {
		return author.Given + " " + author.Family
	}

	return author.Family
}

func firstCrossrefValue(values []string) string {
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

// IsAvailable 检查数据源是否可用
func (c *CrossrefSource) IsAvailable(ctx context.Context) bool {
	_, err := c.client.SearchPapers(ctx, "test", 0, 1)
	return err == nil
}

// GetCapabilities 获取数据源能力
func (c *CrossrefSource) GetCapabilities() SourceCapabilities {
	return SourceCapabilities{
		SupportsFullText:     true,
		SupportsAuthorSearch: true,
		SupportsDateFilter:   true,
		SupportsCitation:     true,
		SupportsPDF:          false,
		RequiresAPIKey:       false,
		SupportedFields:      []string{"title", "author", "container-title", "publisher", "type"},
		MaxResultsPerPage:    1000,
	}
}
