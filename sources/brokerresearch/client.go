package brokerresearch

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const userAgent = "BrokerResearch-MCP-Client/1.0"

var reportYearPattern = regexp.MustCompile(`\b(?:18|19|20)\d{2}\b`)

type BrokerResearchClient struct {
	httpClient  *http.Client
	feeds       []FeedConfig
	apiKey      string
	configError error
}

func NewBrokerResearchClient(feedConfig string, apiKey string, timeout time.Duration) *BrokerResearchClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	feeds, err := parseFeedConfigs(feedConfig)
	return &BrokerResearchClient{
		httpClient:  &http.Client{Timeout: timeout},
		feeds:       feeds,
		apiKey:      strings.TrimSpace(apiKey),
		configError: err,
	}
}

func (client *BrokerResearchClient) SearchReports(ctx context.Context, params *SearchParam) (*SearchResult, error) {
	if client.configError != nil {
		return nil, client.configError
	}
	if len(client.feeds) == 0 {
		return nil, fmt.Errorf("券商研报数据源未配置")
	}
	if params == nil {
		return nil, fmt.Errorf("搜索参数不能为空")
	}

	searchParams := *params
	normalizeSearchParams(&searchParams)

	var allReports []Report
	var failures []string
	for _, feed := range client.feeds {
		reports, err := client.fetchFeedReports(ctx, feed, searchParams)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", feed.Name, err))
			continue
		}
		allReports = append(allReports, reports...)
	}

	if len(allReports) == 0 && len(failures) > 0 {
		return nil, fmt.Errorf("所有券商研报源请求失败: %s", strings.Join(failures, "; "))
	}

	filtered := filterReports(allReports, searchParams)
	sortReports(filtered, searchParams.SortBy, searchParams.SortOrder)
	total := len(filtered)
	paged := paginateReports(filtered, searchParams.Offset, searchParams.Limit)

	return &SearchResult{
		Query:   searchParams.Query,
		Offset:  searchParams.Offset,
		Limit:   searchParams.Limit,
		Total:   total,
		Reports: paged,
	}, nil
}

func (client *BrokerResearchClient) GetReport(ctx context.Context, identifier string) (*Report, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, fmt.Errorf("研报标识符不能为空")
	}

	result, err := client.SearchReports(ctx, &SearchParam{Query: identifier, Limit: MaxLimit})
	if err != nil {
		return nil, err
	}

	for _, report := range result.Reports {
		if reportMatchesIdentifier(report, identifier) {
			return &report, nil
		}
	}

	return nil, fmt.Errorf("未找到标识符为 %s 的券商研报", identifier)
}

func (client *BrokerResearchClient) IsAvailable(ctx context.Context) bool {
	if client.configError != nil || len(client.feeds) == 0 {
		return false
	}

	_, err := client.fetchFeedReports(ctx, client.feeds[0], SearchParam{Limit: 1})
	return err == nil
}

func (client *BrokerResearchClient) fetchFeedReports(ctx context.Context, feed FeedConfig, params SearchParam) ([]Report, error) {
	requestURL := expandFeedURL(feed.URL, params)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "application/json, application/rss+xml, application/atom+xml, text/xml;q=0.9, */*;q=0.8")
	applyAuthHeader(request, client.apiKey, feed)
	for key, value := range feed.Headers {
		if strings.TrimSpace(key) != "" {
			request.Header.Set(key, value)
		}
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return nil, fmt.Errorf("HTTP状态码 %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	format := detectFeedFormat(feed, response.Header.Get("Content-Type"), body)
	switch format {
	case "json":
		return parseJSONReports(body, feed)
	case "rss":
		return parseRSSReports(body, feed)
	case "atom":
		return parseAtomReports(body, feed)
	default:
		return nil, fmt.Errorf("不支持的券商研报源格式: %s", format)
	}
}

func parseFeedConfigs(feedConfig string) ([]FeedConfig, error) {
	feedConfig = strings.TrimSpace(feedConfig)
	if feedConfig == "" {
		return nil, fmt.Errorf("BROKER_RESEARCH_FEEDS 不能为空")
	}

	var feeds []FeedConfig
	if strings.HasPrefix(feedConfig, "[") {
		if err := json.Unmarshal([]byte(feedConfig), &feeds); err != nil {
			return nil, fmt.Errorf("解析 BROKER_RESEARCH_FEEDS JSON数组失败: %w", err)
		}
		return validateFeedConfigs(feeds)
	}

	if strings.HasPrefix(feedConfig, "{") {
		var wrapper struct {
			Feeds []FeedConfig `json:"feeds"`
		}
		if err := json.Unmarshal([]byte(feedConfig), &wrapper); err == nil && len(wrapper.Feeds) > 0 {
			return validateFeedConfigs(wrapper.Feeds)
		}

		var single FeedConfig
		if err := json.Unmarshal([]byte(feedConfig), &single); err != nil {
			return nil, fmt.Errorf("解析 BROKER_RESEARCH_FEEDS JSON对象失败: %w", err)
		}
		return validateFeedConfigs([]FeedConfig{single})
	}

	for _, part := range strings.Split(feedConfig, ",") {
		feedURL := strings.TrimSpace(part)
		if feedURL == "" {
			continue
		}
		feeds = append(feeds, FeedConfig{URL: feedURL})
	}

	return validateFeedConfigs(feeds)
}

func validateFeedConfigs(feeds []FeedConfig) ([]FeedConfig, error) {
	validated := make([]FeedConfig, 0, len(feeds))
	for _, feed := range feeds {
		feed.URL = strings.TrimSpace(feed.URL)
		if feed.URL == "" {
			continue
		}
		if _, err := url.ParseRequestURI(feed.URL); err != nil {
			return nil, fmt.Errorf("无效的券商研报源URL %q: %w", feed.URL, err)
		}
		feed.Name = strings.TrimSpace(feed.Name)
		if feed.Name == "" {
			feed.Name = inferFeedName(feed.URL)
		}
		feed.Format = strings.ToLower(strings.TrimSpace(feed.Format))
		feed.Broker = strings.TrimSpace(feed.Broker)
		feed.Team = strings.TrimSpace(feed.Team)
		validated = append(validated, feed)
	}

	if len(validated) == 0 {
		return nil, fmt.Errorf("BROKER_RESEARCH_FEEDS 未包含有效URL")
	}
	return validated, nil
}

func inferFeedName(feedURL string) string {
	parsedURL, err := url.Parse(feedURL)
	if err != nil || parsedURL.Host == "" {
		return "broker_research"
	}

	return strings.TrimPrefix(strings.ToLower(parsedURL.Host), "www.")
}

func expandFeedURL(feedURL string, params SearchParam) string {
	replacements := map[string]string{
		"{query}":      url.QueryEscape(params.Query),
		"{offset}":     strconv.Itoa(params.Offset),
		"{limit}":      strconv.Itoa(params.Limit),
		"{year}":       url.QueryEscape(params.Year),
		"{year_range}": url.QueryEscape(params.YearRange),
	}
	for placeholder, value := range replacements {
		feedURL = strings.ReplaceAll(feedURL, placeholder, value)
	}

	return feedURL
}

func applyAuthHeader(request *http.Request, globalAPIKey string, feed FeedConfig) {
	if request.Header.Get("Authorization") != "" {
		return
	}

	apiKey := strings.TrimSpace(feed.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(globalAPIKey)
	}
	if apiKey == "" {
		return
	}

	request.Header.Set("Authorization", "Bearer "+apiKey)
}

func detectFeedFormat(feed FeedConfig, contentType string, body []byte) string {
	if feed.Format != "" {
		return feed.Format
	}

	contentType = strings.ToLower(contentType)
	switch {
	case strings.Contains(contentType, "json"):
		return "json"
	case strings.Contains(contentType, "rss"):
		return "rss"
	case strings.Contains(contentType, "atom"):
		return "atom"
	case strings.Contains(contentType, "xml"):
		trimmed := bytes.TrimSpace(body)
		if bytes.Contains(trimmed, []byte("<rss")) {
			return "rss"
		}
		return "atom"
	}

	switch strings.ToLower(filepath.Ext(feed.URL)) {
	case ".json":
		return "json"
	case ".rss":
		return "rss"
	case ".atom":
		return "atom"
	case ".xml":
		trimmed := bytes.TrimSpace(body)
		if bytes.Contains(trimmed, []byte("<rss")) {
			return "rss"
		}
		return "atom"
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return "json"
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return "json"
	}
	if bytes.Contains(trimmed, []byte("<rss")) {
		return "rss"
	}
	return "atom"
}

func parseJSONReports(body []byte, feed FeedConfig) ([]Report, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var payload interface{}
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	reports, err := reportsFromJSONValue(payload, feed)
	if err != nil {
		return nil, err
	}
	return reports, nil
}

func reportsFromJSONValue(value interface{}, feed FeedConfig) ([]Report, error) {
	switch typed := value.(type) {
	case []interface{}:
		return reportsFromJSONArray(typed, feed)
	case map[string]interface{}:
		for _, key := range []string{"items", "reports", "results", "data", "list"} {
			if child, exists := typed[key]; exists {
				reports, err := reportsFromJSONValue(child, feed)
				if err == nil {
					return reports, nil
				}
			}
		}
		if _, hasTitle := typed["title"]; hasTitle {
			report, err := reportFromJSONMap(typed, feed)
			if err != nil {
				return nil, err
			}
			return []Report{report}, nil
		}
	}

	return nil, fmt.Errorf("JSON中未找到研报列表")
}

func reportsFromJSONArray(values []interface{}, feed FeedConfig) ([]Report, error) {
	reports := make([]Report, 0, len(values))
	for _, value := range values {
		item, ok := value.(map[string]interface{})
		if !ok {
			continue
		}
		report, err := reportFromJSONMap(item, feed)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func reportFromJSONMap(item map[string]interface{}, feed FeedConfig) (Report, error) {
	body, err := json.Marshal(item)
	if err != nil {
		return Report{}, err
	}

	var report Report
	if err := json.Unmarshal(body, &report); err != nil {
		return Report{}, err
	}
	report.Raw = item
	applyFeedDefaults(&report, feed)
	return report, nil
}

type rssDocument struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string       `xml:"title"`
	Link        string       `xml:"link"`
	Description string       `xml:"description"`
	PubDate     string       `xml:"pubDate"`
	Author      string       `xml:"author"`
	Creator     string       `xml:"creator"`
	GUID        string       `xml:"guid"`
	Categories  []string     `xml:"category"`
	Enclosure   rssEnclosure `xml:"enclosure"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

func parseRSSReports(body []byte, feed FeedConfig) ([]Report, error) {
	var document rssDocument
	if err := xml.Unmarshal(body, &document); err != nil {
		return nil, fmt.Errorf("解析RSS失败: %w", err)
	}

	if feed.Name == "" && strings.TrimSpace(document.Channel.Title) != "" {
		feed.Name = strings.TrimSpace(document.Channel.Title)
	}

	reports := make([]Report, 0, len(document.Channel.Items))
	for _, item := range document.Channel.Items {
		report := Report{
			ID:            strings.TrimSpace(item.GUID),
			GUID:          strings.TrimSpace(item.GUID),
			Title:         strings.TrimSpace(item.Title),
			Abstract:      strings.TrimSpace(item.Description),
			Author:        firstNonEmpty(item.Creator, item.Author),
			PublishedDate: strings.TrimSpace(item.PubDate),
			URL:           strings.TrimSpace(item.Link),
			Categories:    StringList(item.Categories),
		}
		if strings.Contains(strings.ToLower(item.Enclosure.Type), "pdf") {
			report.PDFURL = strings.TrimSpace(item.Enclosure.URL)
		}
		applyFeedDefaults(&report, feed)
		reports = append(reports, report)
	}

	return reports, nil
}

type atomDocument struct {
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID         string         `xml:"id"`
	Title      string         `xml:"title"`
	Summary    string         `xml:"summary"`
	Content    string         `xml:"content"`
	Published  string         `xml:"published"`
	Updated    string         `xml:"updated"`
	Authors    []atomAuthor   `xml:"author"`
	Links      []atomLink     `xml:"link"`
	Categories []atomCategory `xml:"category"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type atomCategory struct {
	Term  string `xml:"term,attr"`
	Label string `xml:"label,attr"`
}

func parseAtomReports(body []byte, feed FeedConfig) ([]Report, error) {
	var document atomDocument
	if err := xml.Unmarshal(body, &document); err != nil {
		return nil, fmt.Errorf("解析Atom失败: %w", err)
	}

	if feed.Name == "" && strings.TrimSpace(document.Title) != "" {
		feed.Name = strings.TrimSpace(document.Title)
	}

	reports := make([]Report, 0, len(document.Entries))
	for _, entry := range document.Entries {
		report := Report{
			ID:            strings.TrimSpace(entry.ID),
			Title:         strings.TrimSpace(entry.Title),
			Abstract:      firstNonEmpty(entry.Summary, entry.Content),
			Authors:       atomAuthors(entry.Authors),
			PublishedDate: firstNonEmpty(entry.Published, entry.Updated),
			Categories:    atomCategories(entry.Categories),
		}
		report.URL, report.PDFURL = atomLinks(entry.Links)
		applyFeedDefaults(&report, feed)
		reports = append(reports, report)
	}

	return reports, nil
}

func atomAuthors(authors []atomAuthor) StringList {
	values := make([]string, 0, len(authors))
	for _, author := range authors {
		if strings.TrimSpace(author.Name) != "" {
			values = append(values, strings.TrimSpace(author.Name))
		}
	}
	return StringList(values)
}

func atomCategories(categories []atomCategory) StringList {
	values := make([]string, 0, len(categories))
	for _, category := range categories {
		value := firstNonEmpty(category.Label, category.Term)
		if strings.TrimSpace(value) != "" {
			values = append(values, strings.TrimSpace(value))
		}
	}
	return StringList(values)
}

func atomLinks(links []atomLink) (string, string) {
	var pageURL string
	var pdfURL string
	for _, link := range links {
		href := strings.TrimSpace(link.Href)
		if href == "" {
			continue
		}
		linkType := strings.ToLower(link.Type)
		rel := strings.ToLower(link.Rel)
		if pdfURL == "" && (strings.Contains(linkType, "pdf") || strings.Contains(href, ".pdf")) {
			pdfURL = href
			continue
		}
		if pageURL == "" && (rel == "" || rel == "alternate") {
			pageURL = href
		}
	}
	return pageURL, pdfURL
}

func applyFeedDefaults(report *Report, feed FeedConfig) {
	report.ID = strings.TrimSpace(firstNonEmpty(report.ID, report.GUID))
	report.DOI = strings.TrimSpace(report.DOI)
	report.Title = strings.Join(strings.Fields(strings.TrimSpace(report.Title)), " ")
	report.Abstract = strings.TrimSpace(firstNonEmpty(report.Abstract, report.Summary))
	report.Author = strings.TrimSpace(report.Author)
	if len(report.Authors) == 0 && report.Author != "" {
		report.Authors = StringList(splitListText(report.Author))
	}
	report.Broker = strings.TrimSpace(firstNonEmpty(report.Broker, feed.Broker))
	report.Team = strings.TrimSpace(firstNonEmpty(report.Team, feed.Team))
	report.Publisher = strings.TrimSpace(firstNonEmpty(report.Publisher, report.Broker))
	report.SourceName = strings.TrimSpace(firstNonEmpty(report.SourceName, feed.Name))
	report.PublishedDate = NormalizeDate(firstNonEmpty(report.PublishedDate, report.PublishedAt, report.PublishDate, report.Date))
	report.URL = strings.TrimSpace(firstNonEmpty(report.URL, report.Link))
	report.PDFURL = strings.TrimSpace(firstNonEmpty(report.PDFURL, report.PDF))
	if len(report.Categories) == 0 && len(report.Tags) > 0 {
		report.Categories = report.Tags
	}
	report.Language = strings.TrimSpace(report.Language)
	report.Type = strings.TrimSpace(firstNonEmpty(report.Type, "research-report"))
}

func normalizeSearchParams(params *SearchParam) {
	params.Query = normalizeText(params.Query)
	params.Author = normalizeText(params.Author)
	params.Title = normalizeText(params.Title)
	params.Abstract = normalizeText(params.Abstract)
	params.Journal = normalizeText(params.Journal)
	params.Publisher = normalizeText(params.Publisher)
	params.Year = strings.TrimSpace(params.Year)
	params.YearRange = strings.TrimSpace(params.YearRange)
	params.Language = normalizeText(params.Language)
	params.Type = normalizeText(params.Type)
	params.Categories = normalizeStringSlice(params.Categories)
	params.SortBy = strings.TrimSpace(params.SortBy)
	params.SortOrder = strings.TrimSpace(params.SortOrder)

	if params.Limit <= 0 {
		params.Limit = DefaultLimit
	}
	if params.Limit > MaxLimit {
		params.Limit = MaxLimit
	}
	if params.Offset < 0 {
		params.Offset = 0
	}
	if params.SortBy == "" {
		params.SortBy = "published_date"
	}
	if params.SortOrder == "" {
		params.SortOrder = "desc"
	}
}

func filterReports(reports []Report, params SearchParam) []Report {
	filtered := make([]Report, 0, len(reports))
	for _, report := range reports {
		if matchesSearch(report, params) {
			filtered = append(filtered, report)
		}
	}
	return filtered
}

func matchesSearch(report Report, params SearchParam) bool {
	return matchesTextFilters(report, params) &&
		matchesSourceFilters(report, params) &&
		matchesYearFilter(report, params.Year, params.YearRange) &&
		matchesCategoryFilter(report, params.Categories) &&
		matchesAccessFilter(report, params.OpenAccessOnly)
}

func matchesTextFilters(report Report, params SearchParam) bool {
	if params.Query != "" && !containsAllTerms(reportSearchText(report), params.Query) {
		return false
	}
	if params.Title != "" && !containsAllTerms(report.Title, params.Title) {
		return false
	}
	if params.Author != "" && !containsAllTerms(strings.Join(report.Authors, " "), params.Author) {
		return false
	}
	if params.Abstract != "" && !containsAllTerms(report.Abstract, params.Abstract) {
		return false
	}
	return true
}

func matchesSourceFilters(report Report, params SearchParam) bool {
	if params.Publisher != "" && !containsAllTerms(strings.Join([]string{report.Publisher, report.Broker, report.SourceName}, " "), params.Publisher) {
		return false
	}
	if params.Journal != "" && !containsAllTerms(strings.Join([]string{report.Broker, report.Team, report.SourceName}, " "), params.Journal) {
		return false
	}
	if params.Language != "" && !strings.EqualFold(report.Language, params.Language) {
		return false
	}
	if params.Type != "" && !containsAllTerms(report.Type, params.Type) {
		return false
	}
	return true
}

func matchesCategoryFilter(report Report, categories []string) bool {
	return len(categories) == 0 || matchesCategories(report, categories)
}

func matchesAccessFilter(report Report, openAccessOnly bool) bool {
	return !openAccessOnly || report.URL != "" || report.PDFURL != ""
}

func reportSearchText(report Report) string {
	parts := []string{
		report.ID,
		report.GUID,
		report.DOI,
		report.Title,
		report.Abstract,
		strings.Join(report.Authors, " "),
		report.Broker,
		report.Team,
		report.Publisher,
		report.SourceName,
		report.PublishedDate,
		report.URL,
		strings.Join(report.Categories, " "),
		strings.Join(report.Keywords, " "),
		report.Language,
		report.Type,
	}
	return strings.Join(parts, " ")
}

func matchesCategories(report Report, categories []string) bool {
	haystack := strings.Join([]string{
		strings.Join(report.Categories, " "),
		strings.Join(report.Keywords, " "),
		report.Type,
		report.Title,
		report.Abstract,
	}, " ")
	for _, category := range categories {
		if !containsAllTerms(haystack, category) {
			return false
		}
	}
	return true
}

func matchesYearFilter(report Report, year string, yearRange string) bool {
	filter := strings.TrimSpace(firstNonEmpty(yearRange, year))
	if filter == "" {
		return true
	}

	reportYear := reportYear(report)
	if reportYear == 0 {
		return false
	}

	parts := strings.Split(filter, "-")
	if len(parts) == 1 {
		filterYear, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		return err == nil && reportYear == filterYear
	}
	if len(parts) == 2 {
		startYear, startErr := strconv.Atoi(strings.TrimSpace(parts[0]))
		endYear, endErr := strconv.Atoi(strings.TrimSpace(parts[1]))
		if startErr == nil && reportYear < startYear {
			return false
		}
		if endErr == nil && reportYear > endYear {
			return false
		}
		return startErr == nil || endErr == nil
	}

	return strings.Contains(report.PublishedDate, filter)
}

func reportYear(report Report) int {
	match := reportYearPattern.FindString(report.PublishedDate)
	if match == "" {
		return 0
	}
	year, _ := strconv.Atoi(match)
	return year
}

func sortReports(reports []Report, sortBy string, sortOrder string) {
	desc := !strings.EqualFold(sortOrder, "asc")
	sort.SliceStable(reports, func(i, j int) bool {
		comparison := compareReports(reports[i], reports[j], sortBy)
		if desc {
			return comparison > 0
		}
		return comparison < 0
	})
}

func compareReports(left Report, right Report, sortBy string) int {
	switch sortBy {
	case "read_count":
		return compareInts(int(left.ReadCount), int(right.ReadCount))
	case "score", "relevance":
		return compareInts(int(left.Score), int(right.Score))
	case "title":
		return strings.Compare(strings.ToLower(left.Title), strings.ToLower(right.Title))
	default:
		return strings.Compare(left.PublishedDate, right.PublishedDate)
	}
}

func compareInts(left int, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func paginateReports(reports []Report, offset int, limit int) []Report {
	if offset >= len(reports) {
		return []Report{}
	}
	end := offset + limit
	if end > len(reports) {
		end = len(reports)
	}
	return reports[offset:end]
}

func reportMatchesIdentifier(report Report, identifier string) bool {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	candidates := []string{report.ID, report.GUID, report.DOI, report.URL, report.PDFURL, report.Title}
	for _, candidate := range candidates {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate == identifier {
			return true
		}
		if candidate != "" && strings.Contains(candidate, identifier) {
			return true
		}
	}
	return false
}

func containsAllTerms(text string, query string) bool {
	text = strings.ToLower(text)
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 && strings.TrimSpace(query) != "" {
		terms = []string{strings.ToLower(strings.TrimSpace(query))}
	}
	for _, term := range terms {
		if !strings.Contains(text, term) {
			return false
		}
	}
	return true
}

func NormalizeDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	formats := []string{
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02",
		"2006年1月2日",
	}
	for _, format := range formats {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed.Format("2006-01-02")
		}
	}

	if match := reportYearPattern.FindString(value); match != "" {
		return match
	}
	return value
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func normalizeStringSlice(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = normalizeText(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
