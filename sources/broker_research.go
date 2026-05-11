package sources

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Seelly/scholar_mcp_server/common"
	"github.com/Seelly/scholar_mcp_server/sources/brokerresearch"
)

type BrokerResearchSource struct {
	client *brokerresearch.BrokerResearchClient
	config SourceConfig
}

func NewBrokerResearchSource(config SourceConfig) *BrokerResearchSource {
	timeout := time.Duration(config.RequestTimeout) * time.Second
	client := brokerresearch.NewBrokerResearchClient(config.BaseURL, config.APIKey, timeout)

	return &BrokerResearchSource{client: client, config: config}
}

func (source *BrokerResearchSource) GetSourceName() string {
	return "broker_research"
}

func (source *BrokerResearchSource) SearchPapers(ctx context.Context, params common.SearchParams) ([]common.UnifiedPaper, int, error) {
	result, err := source.client.SearchReports(ctx, buildBrokerResearchSearchParam(params))
	if err != nil {
		return nil, 0, err
	}

	papers := make([]common.UnifiedPaper, 0, len(result.Reports))
	for _, report := range result.Reports {
		papers = append(papers, newBrokerResearchUnifiedPaper(report))
	}

	return papers, result.Total, nil
}

func (source *BrokerResearchSource) GetPaper(ctx context.Context, identifier string) (*common.UnifiedPaper, error) {
	report, err := source.client.GetReport(ctx, identifier)
	if err != nil {
		return nil, err
	}

	paper := newBrokerResearchUnifiedPaper(*report)
	return &paper, nil
}

func (source *BrokerResearchSource) IsAvailable(ctx context.Context) bool {
	return source.client.IsAvailable(ctx)
}

func (source *BrokerResearchSource) GetCapabilities() SourceCapabilities {
	return SourceCapabilities{
		SupportsFullText:     true,
		SupportsAuthorSearch: true,
		SupportsDateFilter:   true,
		SupportsCitation:     false,
		SupportsPDF:          true,
		RequiresAPIKey:       false,
		SupportedFields:      []string{"query", "title", "author", "abstract", "publisher", "journal", "year", "year_range", "categories", "language", "type", "open_access"},
		MaxResultsPerPage:    brokerresearch.MaxLimit,
	}
}

func newBrokerResearchUnifiedPaper(report brokerresearch.Report) common.UnifiedPaper {
	doi := strings.TrimSpace(report.DOI)
	paperID := brokerResearchPaperID(report)
	broker := firstNonEmpty(report.Broker, report.Publisher, report.SourceName)
	team := strings.TrimSpace(report.Team)
	journal := firstNonEmpty(team, broker)
	publisher := firstNonEmpty(report.Publisher, report.Broker)
	categories := append([]string(nil), report.Categories...)
	keywords := append([]string(nil), report.Keywords...)

	return common.UnifiedPaper{
		ID:            paperID,
		DOI:           doi,
		Title:         report.Title,
		Abstract:      report.Abstract,
		Authors:       append([]string(nil), report.Authors...),
		Journal:       journal,
		Publisher:     publisher,
		PublishedDate: report.PublishedDate,
		Categories:    categories,
		Keywords:      keywords,
		Language:      report.Language,
		ReadCount:     int(report.ReadCount),
		URL:           report.URL,
		PDFURL:        report.PDFURL,
		IsOpenAccess:  report.URL != "" || report.PDFURL != "",
		Type:          firstNonEmpty(report.Type, "research-report"),
		Source:        "broker_research",
		ExternalIDs: map[string]string{
			"broker_research": paperID,
			"doi":             doi,
		},
		Metadata: map[string]interface{}{
			"broker":        broker,
			"team":          team,
			"feed":          report.SourceName,
			"score":         int(report.Score),
			"source_record": report.Raw,
			"metadata":      report.Metadata,
		},
	}
}

func brokerResearchPaperID(report brokerresearch.Report) string {
	for _, value := range []string{report.ID, report.GUID, report.DOI, report.URL} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	hashInput := fmt.Sprintf("%s|%s|%s|%s", report.SourceName, report.Title, report.PublishedDate, report.PDFURL)
	sum := sha1.Sum([]byte(hashInput))
	return "broker_research:" + hex.EncodeToString(sum[:])
}
