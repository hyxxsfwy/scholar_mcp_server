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
	var (
		result *crossref.SearchResult
		err    error
	)

	if hasCrossrefStructuredFilters(params) {
		result, err = c.client.SearchPapersWithParams(ctx, buildCrossrefSearchParam(params))
	} else {
		result, err = c.client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
	}
	if err != nil {
		return nil, 0, err
	}

	var papers []common.UnifiedPaper
	for _, paper := range result.Papers {
		var authorNames []string
		for _, author := range paper.Author {
			name := ""
			if author.Given != "" && author.Family != "" {
				name = author.Given + " " + author.Family
			} else if author.Family != "" {
				name = author.Family
			}
			if name != "" {
				authorNames = append(authorNames, name)
			}
		}

		title := ""
		if len(paper.Title) > 0 {
			title = paper.Title[0]
		}

		journal := ""
		if len(paper.ContainerTitle) > 0 {
			journal = paper.ContainerTitle[0]
		}

		unified := common.UnifiedPaper{
			ID:            paper.DOI,
			DOI:           paper.DOI,
			Title:         title,
			Authors:       authorNames,
			Journal:       journal,
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
		papers = append(papers, unified)
	}

	return papers, result.TotalResults, nil
}

// GetPaper 获取特定论文
func (c *CrossrefSource) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	paper, err := c.client.GetPaper(ctx, identifier)
	if err != nil {
		return nil, err
	}

	var authorNames []string
	for _, author := range paper.Author {
		name := ""
		if author.Given != "" && author.Family != "" {
			name = author.Given + " " + author.Family
		} else if author.Family != "" {
			name = author.Family
		}
		if name != "" {
			authorNames = append(authorNames, name)
		}
	}

	title := ""
	if len(paper.Title) > 0 {
		title = paper.Title[0]
	}

	journal := ""
	if len(paper.ContainerTitle) > 0 {
		journal = paper.ContainerTitle[0]
	}

	unified := &common.UnifiedPaper{
		ID:            paper.DOI,
		DOI:           paper.DOI,
		Title:         title,
		Authors:       authorNames,
		Journal:       journal,
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
		},
	}

	return unified, nil
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
