package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Seelly/scholar_mcp_server/common"
	openalexapi "github.com/Seelly/scholar_mcp_server/sources/openalex"
	scihubapi "github.com/Seelly/scholar_mcp_server/sources/scihub"
	"github.com/ledongthuc/pdf"
)

const enrichmentUserAgent = "ScholarMCP-Enrichment/1.0"

type SearchResultEnricher struct {
	config     ResultEnrichmentConfig
	httpClient *http.Client
	llmClient  *openAICompatibleLLMClient
}

func NewConfiguredSearchResultEnricher() *SearchResultEnricher {
	return NewSearchResultEnricher(LoadResultEnrichmentConfig())
}

func NewSearchResultEnricher(config ResultEnrichmentConfig) *SearchResultEnricher {
	normalizeResultEnrichmentConfig(&config)
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	client := &http.Client{Timeout: timeout}

	return &SearchResultEnricher{
		config:     config,
		httpClient: client,
		llmClient:  newOpenAICompatibleLLMClient(config.LLM, client),
	}
}

func (enricher *SearchResultEnricher) Enabled() bool {
	return enricher != nil && enricher.config.Enabled && strings.TrimSpace(enricher.config.LLM.APIKey) != ""
}

func (enricher *SearchResultEnricher) EnrichSearchResult(ctx context.Context, result *AggregatedSearchResult) {
	if result == nil {
		return
	}
	enricher.EnrichPapers(ctx, result.Papers)
}

func (enricher *SearchResultEnricher) EnrichPapers(ctx context.Context, papers []common.UnifiedPaper) {
	if !enricher.Enabled() || len(papers) == 0 {
		return
	}

	count := enrichmentPaperCount(len(papers), enricher.config.TopN)
	concurrency := enrichmentConcurrency(count, enricher.config.LLM.MaxConcurrency)
	semaphore := make(chan struct{}, concurrency)
	var waitGroup sync.WaitGroup

	for index := 0; index < count; index++ {
		waitGroup.Add(1)
		go func(paperIndex int) {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			papers[paperIndex].Enrichment = enricher.enrichPaper(ctx, papers[paperIndex])
		}(index)
	}

	waitGroup.Wait()
}

func enrichmentPaperCount(total int, topN int) int {
	if topN <= 0 || topN > total {
		return total
	}
	return topN
}

func enrichmentConcurrency(count int, configured int) int {
	if count <= 0 {
		return 1
	}
	if configured <= 0 || configured > count {
		return count
	}
	return configured
}

func (enricher *SearchResultEnricher) enrichPaper(ctx context.Context, paper common.UnifiedPaper) *common.PaperEnrichment {
	paperCtx, cancel := context.WithTimeout(ctx, time.Duration(enricher.config.TimeoutSeconds)*time.Second)
	defer cancel()

	pdfText, pdfPrefetched, pdfSource, pdfErr := enricher.fetchPDFText(paperCtx, paper)
	inputSources := enrichmentInputSources(paper, pdfText)

	enrichment := &common.PaperEnrichment{
		Model:         enricher.config.LLM.Model,
		GeneratedAt:   time.Now().Format(time.RFC3339),
		InputSources:  inputSources,
		PDFPrefetched: pdfPrefetched,
		PDFSource:     pdfSource,
		PDFTextChars:  len(pdfText),
	}

	// No abstract and no PDF text: skip LLM to avoid hallucinated output.
	if len(inputSources) == 1 && inputSources[0] == "metadata" {
		msg := "跳过LLM生成：无可用摘要和PDF内容"
		if pdfErr != nil {
			msg += "（PDF获取失败: " + pdfErr.Error() + "）"
		}
		enrichment.Error = msg
		return enrichment
	}

	prompt := buildPaperEnrichmentPrompt(paper, pdfText, enricher.config.MaxInputChars)
	content, llmErr := enricher.llmClient.Generate(paperCtx, prompt)
	enrichment.Content = strings.TrimSpace(content)
	if llmErr != nil {
		enrichment.Error = llmErr.Error()
	} else if pdfErr != nil {
		enrichment.Error = "PDF预取失败，已基于摘要生成: " + pdfErr.Error()
	}

	return enrichment
}

func enrichmentInputSources(paper common.UnifiedPaper, pdfText string) []string {
	var sources []string
	if strings.TrimSpace(paper.Abstract) != "" {
		sources = append(sources, "abstract")
	}
	if strings.TrimSpace(pdfText) != "" {
		sources = append(sources, "pdf_text")
	}
	if len(sources) == 0 {
		sources = append(sources, "metadata")
	}
	return sources
}

func (enricher *SearchResultEnricher) fetchPDFText(ctx context.Context, paper common.UnifiedPaper) (string, bool, string, error) {
	if !enricher.config.PrefetchPDF {
		return "", false, "", nil
	}

	candidates := enricher.pdfCandidates(ctx, paper)
	var failures []string
	for _, candidate := range candidates {
		text, err := enricher.fetchPDFCandidateText(ctx, candidate)
		if err == nil {
			return text, true, candidate.Source, nil
		}
		failures = append(failures, fmt.Sprintf("%s: %v", candidate.Source, err))
	}
	if len(failures) > 0 {
		return "", false, "", errors.New(strings.Join(failures, "; "))
	}

	return "", false, "", nil
}

type pdfCandidate struct {
	URL    string
	Source string
}

func (enricher *SearchResultEnricher) pdfCandidates(ctx context.Context, paper common.UnifiedPaper) []pdfCandidate {
	var candidates []pdfCandidate
	if strings.TrimSpace(paper.PDFURL) != "" {
		candidates = append(candidates, pdfCandidate{URL: paper.PDFURL, Source: "result_pdf_url"})
	}
	switch enricher.config.PDFFallback {
	case "open_access", "scihub":
		if fallbackURL := enricher.resolveOpenAccessPDFURL(ctx, paper); fallbackURL != "" && fallbackURL != strings.TrimSpace(paper.PDFURL) {
			candidates = append(candidates, pdfCandidate{URL: fallbackURL, Source: "openalex_open_access"})
		}
		if enricher.config.PDFFallback == "scihub" {
			if doi := paperDOI(paper); doi != "" {
				if sciHubURL := enricher.resolveSciHubPDFURL(ctx, doi); sciHubURL != "" {
					candidates = append(candidates, pdfCandidate{URL: sciHubURL, Source: "scihub"})
				}
			}
		}
	}
	return candidates
}

func (enricher *SearchResultEnricher) resolveSciHubPDFURL(ctx context.Context, doi string) string {
	client := scihubapi.NewSciHubClient()
	response, err := client.GetPDFURL(ctx, doi)
	if err != nil || !response.Success || strings.TrimSpace(response.PDFURL) == "" {
		return ""
	}
	return strings.TrimSpace(response.PDFURL)
}

func (enricher *SearchResultEnricher) fetchPDFCandidateText(ctx context.Context, candidate pdfCandidate) (string, error) {
	requestURL := strings.TrimSpace(candidate.URL)
	if !isHTTPURL(requestURL) {
		return "", fmt.Errorf("非HTTP(S) PDF链接")
	}

	data, err := enricher.downloadPDF(ctx, requestURL)
	if err != nil {
		return "", err
	}
	text, err := extractPDFText(data, enricher.config.PDFTextMaxChars)
	if err != nil {
		return "", err
	}

	return text, nil
}

func (enricher *SearchResultEnricher) resolveOpenAccessPDFURL(ctx context.Context, paper common.UnifiedPaper) string {
	doi := paperDOI(paper)
	if doi == "" {
		return ""
	}
	settings := openAlexResolverSettings()
	client := openalexapi.NewOpenAlexClient(settings.APIKey)
	if settings.BaseURL != "" {
		client = openalexapi.NewOpenAlexClientWithBaseURL(settings.APIKey, settings.BaseURL)
	}

	openAlexPaper, err := client.GetPaper(ctx, doi)
	if err != nil {
		return ""
	}
	return firstOpenAlexResolvedPDFURL(*openAlexPaper)
}

func paperDOI(paper common.UnifiedPaper) string {
	if strings.TrimSpace(paper.DOI) != "" {
		return strings.TrimSpace(paper.DOI)
	}
	if paper.ExternalIDs != nil {
		return strings.TrimSpace(paper.ExternalIDs["doi"])
	}
	return ""
}

func openAlexResolverSettings() SourceConfig {
	settings := SourceConfig{
		APIKey:  firstEnvValue("OPENALEX_EMAIL", "CROSSREF_EMAIL"),
		BaseURL: strings.TrimSpace(firstEnvValue("OPENALEX_BASE_URL")),
	}
	projectConfig, _, err := LoadProjectConfig()
	if err == nil && projectConfig != nil {
		if patch, exists := projectConfig.Sources["openalex"]; exists {
			settings = patch.apply(settings)
		}
	}
	return settings
}

func firstOpenAlexResolvedPDFURL(paper openalexapi.Paper) string {
	if paper.PrimaryLocation.IsOA && paper.PrimaryLocation.PDFURL != "" {
		return paper.PrimaryLocation.PDFURL
	}
	if paper.BestOALocation != nil && paper.BestOALocation.IsOA && paper.BestOALocation.PDFURL != "" {
		return paper.BestOALocation.PDFURL
	}
	for _, location := range paper.Locations {
		if location.IsOA && location.PDFURL != "" {
			return location.PDFURL
		}
	}
	return ""
}

func (enricher *SearchResultEnricher) downloadPDF(ctx context.Context, pdfURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, pdfURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", enrichmentUserAgent)
	request.Header.Set("Accept", "application/pdf,*/*;q=0.8")

	response, err := enricher.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("下载PDF失败，状态码: %d", response.StatusCode)
	}

	reader := io.LimitReader(response.Body, enricher.config.MaxPDFBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > enricher.config.MaxPDFBytes {
		return nil, fmt.Errorf("PDF超过大小限制 %d 字节", enricher.config.MaxPDFBytes)
	}

	return data, nil
}

func extractPDFText(data []byte, maxChars int) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("解析PDF失败: %w", err)
	}

	var builder strings.Builder
	for pageIndex := 1; pageIndex <= reader.NumPage(); pageIndex++ {
		page := reader.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		builder.WriteString(text)
		if maxChars > 0 && builder.Len() >= maxChars {
			return common.TruncateText(builder.String(), maxChars), nil
		}
	}

	return strings.TrimSpace(builder.String()), nil
}

func buildPaperEnrichmentPrompt(paper common.UnifiedPaper, pdfText string, maxInputChars int) string {
	var builder strings.Builder
	builder.WriteString("请基于以下论文元数据、摘要和可用PDF文本，生成中文文献大纲与较详细梗概。\n")
	builder.WriteString("要求：\n")
	builder.WriteString("1. 只根据提供材料总结，不要编造未出现的信息。\n")
	builder.WriteString("2. 输出结构包含：研究问题、方法/数据、主要发现、贡献、局限、适合继续追问的问题。\n")
	builder.WriteString("3. 内容面向将继续调用MCP工具的Agent，语言清晰、信息密度高。\n\n")
	builder.WriteString("论文信息：\n")
	writePromptField(&builder, "标题", paper.Title)
	writePromptField(&builder, "作者", strings.Join(paper.Authors, ", "))
	writePromptField(&builder, "来源", paper.Source)
	writePromptField(&builder, "期刊/会议", paper.Journal)
	writePromptField(&builder, "发表日期", paper.PublishedDate)
	writePromptField(&builder, "DOI", paper.DOI)
	writePromptField(&builder, "分类", strings.Join(paper.Categories, ", "))
	writePromptField(&builder, "摘要", paper.Abstract)
	if strings.TrimSpace(pdfText) != "" {
		writePromptField(&builder, "PDF文本节选", pdfText)
	}

	return common.TruncateText(builder.String(), maxInputChars)
}

func writePromptField(builder *strings.Builder, name string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	builder.WriteString(name)
	builder.WriteString(": ")
	builder.WriteString(value)
	builder.WriteString("\n")
}

func isHTTPURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return parsedURL.Scheme == "http" || parsedURL.Scheme == "https"
}

type openAICompatibleLLMClient struct {
	config     LLMConfig
	httpClient *http.Client
}

func newOpenAICompatibleLLMClient(config LLMConfig, httpClient *http.Client) *openAICompatibleLLMClient {
	return &openAICompatibleLLMClient{config: config, httpClient: httpClient}
}

func (client *openAICompatibleLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	return client.GenerateWithSystem(ctx, "你是严谨的学术研究助理，擅长从论文摘要和PDF节选中提炼结构化中文文献大纲。", prompt)
}

func (client *openAICompatibleLLMClient) GenerateWithSystem(ctx context.Context, systemPrompt string, prompt string) (string, error) {
	requestBody := openAIChatCompletionRequest{
		Model:       client.config.Model,
		MaxTokens:   client.config.MaxTokens,
		Temperature: client.config.Temperature,
		Messages: []openAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionURL(client.config.BaseURL), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+client.config.APIKey)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("LLM API请求失败，状态码: %d, 响应: %s", response.StatusCode, common.TruncateText(string(responseBody), 500))
	}

	var completion openAIChatCompletionResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return "", err
	}
	if completion.Error.Message != "" {
		return "", fmt.Errorf("LLM API错误: %s", completion.Error.Message)
	}
	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("LLM API未返回内容")
	}

	return completion.Choices[0].Message.Content, nil
}

func chatCompletionURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	return baseURL + "/chat/completions"
}

type openAIChatCompletionRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatCompletionResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
