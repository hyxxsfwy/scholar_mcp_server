package sources

import (
	"context"
	"github.com/Seelly/scholar_mcp_server/sources/scihub"

	"github.com/Seelly/scholar_mcp_server/common"
)

// SciHubSource Sci-Hub数据源实现
type SciHubSource struct {
	client *scihub.SciHubClient
	config SourceConfig
}

// NewSciHubSource 创建新的Sci-Hub数据源
func NewSciHubSource(config SourceConfig) *SciHubSource {
	return &SciHubSource{
		client: scihub.NewSciHubClient(),
		config: config,
	}
}

// GetSourceName 获取数据源名称
func (s *SciHubSource) GetSourceName() string {
	return "scihub"
}

// SearchPapers 搜索论文
func (s *SciHubSource) SearchPapers(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	result, err := s.client.SearchPapers(ctx, params.Query, params.Limit)
	if err != nil {
		return nil, 0, err
	}

	var papers []common.UnifiedPaper
	for _, paper := range result.Papers {
		var authors []string
		if paper.Authors != "" {
			authors = common.SplitAuthors(paper.Authors)
		}

		unified := common.UnifiedPaper{
			ID:            paper.DOI,
			DOI:           paper.DOI,
			Title:         paper.Title,
			Authors:       authors,
			Journal:       paper.Journal,
			PublishedDate: paper.Year,
			PDFURL:        paper.PDFURL,
			IsOpenAccess:  paper.Available,
			Type:          "article",
			Source:        "scihub",
			ExternalIDs: map[string]string{
				"doi": paper.DOI,
			},
			Metadata: map[string]interface{}{
				"scihub_url": paper.SciHubURL,
				"file_size":  paper.FileSize,
				"citation":   paper.Citation,
			},
		}
		papers = append(papers, unified)
	}

	return papers, result.Total, nil
}

// GetPaper 获取特定论文
func (s *SciHubSource) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	paper, err := s.client.GetPaper(ctx, identifier)
	if err != nil {
		return nil, err
	}

	var authors []string
	if paper.Authors != "" {
		authors = common.SplitAuthors(paper.Authors)
	}

	unified := &common.UnifiedPaper{
		ID:            paper.DOI,
		DOI:           paper.DOI,
		Title:         paper.Title,
		Authors:       authors,
		Journal:       paper.Journal,
		PublishedDate: paper.Year,
		PDFURL:        paper.PDFURL,
		IsOpenAccess:  paper.Available,
		Type:          "article",
		Source:        "scihub",
		ExternalIDs: map[string]string{
			"doi": paper.DOI,
		},
		Metadata: map[string]interface{}{
			"scihub_url": paper.SciHubURL,
			"file_size":  paper.FileSize,
			"citation":   paper.Citation,
		},
	}

	return unified, nil
}

// IsAvailable 检查数据源是否可用
func (s *SciHubSource) IsAvailable(ctx context.Context) bool {
	// Sci-Hub可用性检查比较复杂，这里简化处理
	status, err := s.client.GetMirrorStatus(ctx)
	if err != nil {
		return false
	}
	return status.ActiveCount > 0
}

// GetCapabilities 获取数据源能力
func (s *SciHubSource) GetCapabilities() SourceCapabilities {
	return SourceCapabilities{
		SupportsFullText:     false,
		SupportsAuthorSearch: false,
		SupportsDateFilter:   false,
		SupportsCitation:     false,
		SupportsPDF:          true, // 这是Sci-Hub的主要功能
		RequiresAPIKey:       false,
		SupportedFields:      []string{"doi", "title"},
		MaxResultsPerPage:    50,
	}
}
