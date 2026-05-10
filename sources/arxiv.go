package sources

import (
	"context"

	"github.com/Seelly/scholar_mcp_server/common"
	"github.com/Seelly/scholar_mcp_server/sources/arxiv"
)

// ArxivSource arXiv数据源实现
type ArxivSource struct {
	client *arxiv.ArxivClient
	config SourceConfig
}

// NewArxivSource 创建新的arXiv数据源
func NewArxivSource(config SourceConfig) *ArxivSource {
	return &ArxivSource{
		client: arxiv.NewArxivClient(),
		config: config,
	}
}

// GetSourceName 获取数据源名称
func (a *ArxivSource) GetSourceName() string {
	return "arxiv"
}

// SearchPapers 搜索论文
func (a *ArxivSource) SearchPapers(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	query := params.Query
	if hasArxivStructuredFilters(params) {
		query = buildArxivSearchQuery(params)
	}

	result, err := a.client.SearchPapers(ctx, query, params.Offset, params.Limit)
	if err != nil {
		return nil, 0, err
	}

	var papers []common.UnifiedPaper
	for _, paper := range result.Papers {
		unified := common.UnifiedPaper{
			ID:            paper.ID,
			DOI:           paper.DOI,
			Title:         paper.Title,
			Abstract:      paper.Abstract,
			Authors:       paper.Authors,
			Journal:       "", // arXiv没有期刊概念
			PublishedDate: paper.Published,
			Categories:    paper.Categories,
			URL:           paper.URL,
			PDFURL:        paper.PDFURL,
			IsOpenAccess:  true, // arXiv都是开放获取
			Type:          "preprint",
			Source:        "arxiv",
			ExternalIDs: map[string]string{
				"arxiv": paper.ID,
				"doi":   paper.DOI,
			},
			Metadata: map[string]interface{}{
				"journal_ref": paper.JournalRef,
				"comment":     paper.Comment,
				"updated":     paper.Updated,
			},
		}
		papers = append(papers, unified)
	}

	return papers, result.TotalCount, nil
}

// GetPaper 获取特定论文
func (a *ArxivSource) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	paper, err := a.client.GetPaper(ctx, identifier)
	if err != nil {
		return nil, err
	}

	unified := &common.UnifiedPaper{
		ID:            paper.ID,
		DOI:           paper.DOI,
		Title:         paper.Title,
		Abstract:      paper.Abstract,
		Authors:       paper.Authors,
		PublishedDate: paper.Published,
		Categories:    paper.Categories,
		URL:           paper.URL,
		PDFURL:        paper.PDFURL,
		IsOpenAccess:  true,
		Type:          "preprint",
		Source:        "arxiv",
		ExternalIDs: map[string]string{
			"arxiv": paper.ID,
			"doi":   paper.DOI,
		},
		Metadata: map[string]interface{}{
			"journal_ref": paper.JournalRef,
			"comment":     paper.Comment,
			"updated":     paper.Updated,
		},
	}

	return unified, nil
}

// IsAvailable 检查数据源是否可用
func (a *ArxivSource) IsAvailable(ctx context.Context) bool {
	// 简单的连通性检查
	_, err := a.client.SearchPapers(ctx, "test", 0, 1)
	return err == nil
}

// GetCapabilities 获取数据源能力
func (a *ArxivSource) GetCapabilities() SourceCapabilities {
	return SourceCapabilities{
		SupportsFullText:     true,
		SupportsAuthorSearch: true,
		SupportsDateFilter:   true,
		SupportsCitation:     false,
		SupportsPDF:          true,
		RequiresAPIKey:       false,
		SupportedFields:      []string{"title", "author", "abstract", "category", "comment"},
		MaxResultsPerPage:    2000,
	}
}
