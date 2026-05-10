package openalex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	OpenAlexAPIBaseURL = "https://api.openalex.org/works"
	DefaultLimit       = 10
	MaxLimit           = 200
)

type OpenAlexClient struct {
	httpClient *http.Client
	baseURL    string
	email      string
}

func NewOpenAlexClient(email string) *OpenAlexClient {
	return &OpenAlexClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    OpenAlexAPIBaseURL,
		email:      strings.TrimSpace(email),
	}
}

func NewOpenAlexClientWithBaseURL(email, baseURL string) *OpenAlexClient {
	client := NewOpenAlexClient(email)
	if strings.TrimSpace(baseURL) != "" {
		client.baseURL = strings.TrimSpace(baseURL)
	}

	return client
}

func (client *OpenAlexClient) SearchPapers(ctx context.Context, query string, offset, limit int) (*SearchResult, error) {
	return client.SearchPapersWithParams(ctx, &OpenAlexSearchParam{Query: query, Offset: offset, Limit: limit})
}

func (client *OpenAlexClient) SearchPapersWithParams(ctx context.Context, params *OpenAlexSearchParam) (*SearchResult, error) {
	if params == nil {
		return nil, fmt.Errorf("搜索参数不能为空")
	}
	if strings.TrimSpace(params.Query) == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}

	searchParams := *params
	normalizeSearchParams(&searchParams)

	papers, total, err := client.searchPaged(ctx, searchParams)
	if err != nil {
		return nil, err
	}

	return &SearchResult{
		Query:  searchParams.Query,
		Offset: searchParams.Offset,
		Limit:  searchParams.Limit,
		Total:  total,
		Papers: papers,
	}, nil
}

func (client *OpenAlexClient) GetPaper(ctx context.Context, identifier string) (*Paper, error) {
	identifier = normalizeIdentifier(identifier)
	if identifier == "" {
		return nil, fmt.Errorf("论文标识符不能为空")
	}

	requestURL := fmt.Sprintf("%s/%s", client.baseURL, url.PathEscape(identifier))
	if client.email != "" {
		queryValues := url.Values{}
		queryValues.Set("mailto", client.email)
		requestURL += "?" + queryValues.Encode()
	}

	body, err := client.get(ctx, requestURL)
	if err != nil {
		return nil, err
	}

	var paper Paper
	if err := json.Unmarshal(body, &paper); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	return &paper, nil
}

func (client *OpenAlexClient) searchPaged(ctx context.Context, params OpenAlexSearchParam) ([]Paper, int, error) {
	perPage := calculatePerPage(params.Offset, params.Limit)
	page := params.Offset/perPage + 1
	skip := params.Offset - ((page - 1) * perPage)
	remaining := params.Limit
	total := 0
	var papers []Paper

	for remaining > 0 {
		apiResponse, err := client.searchPage(ctx, params, page, perPage)
		if err != nil {
			return nil, 0, err
		}
		if total == 0 {
			total = apiResponse.Meta.Count
		}

		results := apiResponse.Results
		if skip > 0 {
			if skip >= len(results) {
				return papers, total, nil
			}
			results = results[skip:]
			skip = 0
		}

		if len(results) > remaining {
			results = results[:remaining]
		}
		papers = append(papers, results...)
		remaining -= len(results)
		if len(apiResponse.Results) < perPage || len(results) == 0 {
			break
		}
		page++
	}

	return papers, total, nil
}

func (client *OpenAlexClient) searchPage(ctx context.Context, params OpenAlexSearchParam, page, perPage int) (*APIResponse, error) {
	requestURL := client.buildSearchURL(params, page, perPage)
	body, err := client.get(ctx, requestURL)
	if err != nil {
		return nil, err
	}

	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}
	if apiResponse.Error != "" {
		return nil, fmt.Errorf("OpenAlex API错误: %s", apiResponse.Error)
	}

	return &apiResponse, nil
}

func (client *OpenAlexClient) get(ctx context.Context, requestURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	request.Header.Set("User-Agent", "OpenAlex-MCP-Client/1.0")

	response, err := doWithRetry(ctx, client.httpClient, request)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("OpenAlex未找到请求资源")
	}
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return body, nil
}

func (client *OpenAlexClient) buildSearchURL(params OpenAlexSearchParam, page, perPage int) string {
	queryValues := url.Values{}
	queryValues.Set("search", params.Query)
	queryValues.Set("page", strconv.Itoa(page))
	queryValues.Set("per-page", strconv.Itoa(perPage))
	queryValues.Set("select", "id,doi,title,display_name,publication_year,publication_date,type,type_crossref,cited_by_count,ids,authorships,primary_location,best_oa_location,locations,open_access,biblio,concepts,keywords,abstract_inverted_index")
	if client.email != "" {
		queryValues.Set("mailto", client.email)
	}
	if filter := buildFilter(params); filter != "" {
		queryValues.Set("filter", filter)
	}
	if sort := buildSort(params.SortBy, params.SortOrder); sort != "" {
		queryValues.Set("sort", sort)
	}

	return fmt.Sprintf("%s?%s", client.baseURL, queryValues.Encode())
}

func normalizeSearchParams(params *OpenAlexSearchParam) {
	params.Query = strings.Join(strings.Fields(params.Query), " ")
	params.Year = strings.TrimSpace(params.Year)
	params.YearRange = strings.TrimSpace(params.YearRange)
	params.Type = strings.TrimSpace(params.Type)
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
}

func calculatePerPage(offset, limit int) int {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if offset < 0 {
		offset = 0
	}

	perPage := limit
	if remainder := offset % limit; remainder > 0 {
		perPage = limit + remainder
	}
	if perPage > MaxLimit {
		perPage = MaxLimit
	}

	return perPage
}

func buildFilter(params OpenAlexSearchParam) string {
	var filters []string
	if fromDate, untilDate := buildDateFilter(params.YearRange, params.Year); fromDate != "" && untilDate != "" {
		filters = append(filters, "from_publication_date:"+fromDate, "to_publication_date:"+untilDate)
	}
	if params.Type != "" {
		filters = append(filters, "type:"+params.Type)
	}
	if params.OpenAccessOnly {
		filters = append(filters, "is_oa:true")
	}
	if params.MinCitations > 0 {
		filters = append(filters, fmt.Sprintf("cited_by_count:>%d", params.MinCitations))
	}

	return strings.Join(filters, ",")
}

func buildDateFilter(yearRange, year string) (string, string) {
	yearFilter := strings.TrimSpace(yearRange)
	if yearFilter == "" {
		yearFilter = strings.TrimSpace(year)
	}
	if yearFilter == "" {
		return "", ""
	}

	parts := strings.Split(yearFilter, "-")
	if len(parts) == 1 && len(parts[0]) == 4 {
		return parts[0] + "-01-01", parts[0] + "-12-31"
	}
	if len(parts) == 2 && len(parts[0]) == 4 && len(parts[1]) == 4 {
		return parts[0] + "-01-01", parts[1] + "-12-31"
	}

	return "", ""
}

func buildSort(sortBy, sortOrder string) string {
	if sortOrder == "" {
		sortOrder = "desc"
	}
	switch sortBy {
	case "citation_count", "citations":
		return "cited_by_count:" + sortOrder
	case "published_date", "date":
		return "publication_date:" + sortOrder
	case "title":
		return "display_name:" + sortOrder
	default:
		return ""
	}
}

func normalizeIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return ""
	}
	if doi := extractDOI(identifier); doi != "" {
		return "doi:" + doi
	}
	identifier = strings.TrimPrefix(identifier, "https://openalex.org/")
	identifier = strings.TrimPrefix(identifier, "http://openalex.org/")
	identifier = strings.TrimPrefix(identifier, "https://api.openalex.org/works/")
	identifier = strings.TrimPrefix(identifier, "http://api.openalex.org/works/")

	return identifier
}

func extractDOI(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "doi:") {
		return strings.TrimSpace(value[4:])
	}
	for _, prefix := range []string{"https://doi.org/", "http://doi.org/", "https://dx.doi.org/", "http://dx.doi.org/"} {
		if strings.HasPrefix(strings.ToLower(value), prefix) {
			return strings.TrimSpace(value[len(prefix):])
		}
	}
	if strings.HasPrefix(value, "10.") {
		return value
	}

	return ""
}

func doWithRetry(ctx context.Context, httpClient *http.Client, request *http.Request) (*http.Response, error) {
	var response *http.Response
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		response, err = httpClient.Do(request.Clone(ctx))
		if err != nil || response.StatusCode != http.StatusTooManyRequests {
			return response, err
		}
		response.Body.Close()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(1<<uint(attempt)) * time.Second):
		}
	}

	return response, err
}
