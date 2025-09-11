package sources

import (
	"context"
	"fmt"
	"github.com/Seelly/scholar_mcp_server/sources/adsabs"

	"github.com/Seelly/scholar_mcp_server/common"
)

// AdsabsSource ADSABS数据源实现
type AdsabsSource struct {
	client *adsabs.AdsabsClient
	config SourceConfig
}

// NewAdsabsSource 创建新的ADSABS数据源
func NewAdsabsSource(config SourceConfig) *AdsabsSource {
	if config.APIKey == "" {
		return nil // ADSABS需要API密钥
	}

	return &AdsabsSource{
		client: adsabs.NewAdsabsClient(config.APIKey),
		config: config,
	}
}

// GetSourceName 获取数据源名称
func (a *AdsabsSource) GetSourceName() string {
	return "adsabs"
}

// SearchPapers 搜索论文
func (a *AdsabsSource) SearchPapers(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	if a.client == nil {
		return nil, 0, fmt.Errorf("ADSABS client not initialized - API key required")
	}

	result, err := a.client.SearchPapers(ctx, params.Query, params.Offset, params.Limit)
	if err != nil {
		return nil, 0, err
	}

	var papers []common.UnifiedPaper
	for _, paper := range result.Papers {
		title := ""
		if len(paper.Title) > 0 {
			title = paper.Title[0]
		}

		doi := ""
		if len(paper.DOI) > 0 {
			doi = paper.DOI[0]
		}

		arxivID := ""
		if len(paper.ArxivID) > 0 {
			arxivID = paper.ArxivID[0]
		}

		unified := common.UnifiedPaper{
			ID:            paper.Bibcode,
			DOI:           doi,
			Title:         title,
			Abstract:      paper.Abstract,
			Authors:       paper.Author,
			Journal:       paper.Pub,
			PublishedDate: paper.PubDate,
			Volume:        paper.Volume,
			Issue:         paper.Issue,
			Categories:    paper.Keyword,
			CitationCount: paper.CitationCount,
			ReadCount:     paper.ReadCount,
			Type:          paper.DocType,
			Source:        "adsabs",
			ExternalIDs: map[string]string{
				"adsabs": paper.Bibcode,
				"doi":    doi,
				"arxiv":  arxivID,
			},
			Metadata: map[string]interface{}{
				"bibstem":    paper.Bibstem,
				"property":   paper.Property,
				"database":   paper.Database,
				"facility":   paper.Facility,
				"instrument": paper.Instrument,
				"object":     paper.Object,
				"orcid":      paper.ORCID,
				"grant":      paper.Grant,
				"aff":        paper.Aff,
			},
		}
		papers = append(papers, unified)
	}

	return papers, result.NumFound, nil
}

// GetPaper 获取特定论文
func (a *AdsabsSource) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	if a.client == nil {
		return nil, fmt.Errorf("ADSABS client not initialized - API key required")
	}

	paper, err := a.client.GetPaper(ctx, identifier)
	if err != nil {
		return nil, err
	}

	title := ""
	if len(paper.Title) > 0 {
		title = paper.Title[0]
	}

	doi := ""
	if len(paper.DOI) > 0 {
		doi = paper.DOI[0]
	}

	arxivID := ""
	if len(paper.ArxivID) > 0 {
		arxivID = paper.ArxivID[0]
	}

	unified := &common.UnifiedPaper{
		ID:            paper.Bibcode,
		DOI:           doi,
		Title:         title,
		Abstract:      paper.Abstract,
		Authors:       paper.Author,
		Journal:       paper.Pub,
		PublishedDate: paper.PubDate,
		Volume:        paper.Volume,
		Issue:         paper.Issue,
		Categories:    paper.Keyword,
		CitationCount: paper.CitationCount,
		ReadCount:     paper.ReadCount,
		Type:          paper.DocType,
		Source:        "adsabs",
		ExternalIDs: map[string]string{
			"adsabs": paper.Bibcode,
			"doi":    doi,
			"arxiv":  arxivID,
		},
		Metadata: map[string]interface{}{
			"bibstem":    paper.Bibstem,
			"property":   paper.Property,
			"database":   paper.Database,
			"facility":   paper.Facility,
			"instrument": paper.Instrument,
			"object":     paper.Object,
			"orcid":      paper.ORCID,
			"grant":      paper.Grant,
			"aff":        paper.Aff,
		},
	}

	return unified, nil
}

// IsAvailable 检查数据源是否可用
func (a *AdsabsSource) IsAvailable(ctx context.Context) bool {
	if a.client == nil {
		return false
	}
	_, err := a.client.SearchPapers(ctx, "test", 0, 1)
	return err == nil
}

// GetCapabilities 获取数据源能力
func (a *AdsabsSource) GetCapabilities() SourceCapabilities {
	return SourceCapabilities{
		SupportsFullText:     true,
		SupportsAuthorSearch: true,
		SupportsDateFilter:   true,
		SupportsCitation:     true,
		SupportsPDF:          false,
		RequiresAPIKey:       true,
		SupportedFields:      []string{"title", "author", "abstract", "bibcode", "object", "year"},
		MaxResultsPerPage:    2000,
	}
}
