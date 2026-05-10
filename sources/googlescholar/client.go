package googlescholar

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
	GoogleScholarAPIBaseURL = "https://serpapi.com/search.json"
	DefaultLimit            = 10
	MaxLimit                = 20
)

type GoogleScholarClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

func NewGoogleScholarClient(apiKey string) *GoogleScholarClient {
	return &GoogleScholarClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    GoogleScholarAPIBaseURL,
		apiKey:     strings.TrimSpace(apiKey),
	}
}

func NewGoogleScholarClientWithBaseURL(apiKey, baseURL string) *GoogleScholarClient {
	client := NewGoogleScholarClient(apiKey)
	if strings.TrimSpace(baseURL) != "" {
		client.baseURL = strings.TrimSpace(baseURL)
	}

	return client
}

func (client *GoogleScholarClient) SearchPapers(ctx context.Context, query string, offset, limit int) (*SearchResult, error) {
	return client.SearchPapersWithParams(ctx, &GoogleScholarSearchParam{
		Query:  query,
		Offset: offset,
		Limit:  limit,
	})
}

func (client *GoogleScholarClient) SearchPapersWithParams(ctx context.Context, params *GoogleScholarSearchParam) (*SearchResult, error) {
	if params == nil {
		return nil, fmt.Errorf("搜索参数不能为空")
	}
	if strings.TrimSpace(params.Query) == "" {
		return nil, fmt.Errorf("查询关键词不能为空")
	}
	if client.apiKey == "" {
		return nil, fmt.Errorf("Google Scholar SerpAPI API key不能为空")
	}

	searchParams := *params
	normalizeSearchParams(&searchParams)

	requestURL := client.buildSearchURL(searchParams)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	request.Header.Set("User-Agent", "GoogleScholar-MCP-Client/1.0")
	response, err := doWithRetry(ctx, client.httpClient, request)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("API请求失败，状态码: %d, 响应: %s", response.StatusCode, string(body))
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}
	if apiResponse.Error != "" {
		return nil, fmt.Errorf("Google Scholar API错误: %s", apiResponse.Error)
	}

	totalResults := apiResponse.SearchInformation.TotalResults
	if totalResults == 0 {
		totalResults = len(apiResponse.OrganicResults)
	}

	return &SearchResult{
		Query:  searchParams.Query,
		Offset: searchParams.Offset,
		Limit:  searchParams.Limit,
		Total:  totalResults,
		Papers: apiResponse.OrganicResults,
	}, nil
}

func (client *GoogleScholarClient) buildSearchURL(params GoogleScholarSearchParam) string {
	queryValues := url.Values{}
	queryValues.Set("engine", "google_scholar")
	queryValues.Set("q", params.Query)
	queryValues.Set("api_key", client.apiKey)
	queryValues.Set("start", strconv.Itoa(params.Offset))
	queryValues.Set("num", strconv.Itoa(params.Limit))

	if params.Author != "" {
		queryValues.Set("as_sauthors", params.Author)
	}
	if params.Journal != "" {
		queryValues.Set("as_publication", params.Journal)
	}
	applyYearFilter(queryValues, params.YearRange, params.Year)
	if params.SortBy == "date" || params.SortBy == "published_date" {
		queryValues.Set("scisbd", "1")
	}

	return fmt.Sprintf("%s?%s", client.baseURL, queryValues.Encode())
}

func normalizeSearchParams(params *GoogleScholarSearchParam) {
	params.Query = strings.Join(strings.Fields(params.Query), " ")
	params.Author = strings.Join(strings.Fields(params.Author), " ")
	params.Journal = strings.Join(strings.Fields(params.Journal), " ")
	params.Year = strings.TrimSpace(params.Year)
	params.YearRange = strings.TrimSpace(params.YearRange)
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

func applyYearFilter(queryValues url.Values, yearRange, year string) {
	yearFilter := strings.TrimSpace(yearRange)
	if yearFilter == "" {
		yearFilter = strings.TrimSpace(year)
	}
	if yearFilter == "" {
		return
	}

	parts := strings.Split(yearFilter, "-")
	if len(parts) == 1 {
		trimmedYear := strings.TrimSpace(parts[0])
		queryValues.Set("as_ylo", trimmedYear)
		queryValues.Set("as_yhi", trimmedYear)
		return
	}
	if len(parts) == 2 {
		fromYear := strings.TrimSpace(parts[0])
		toYear := strings.TrimSpace(parts[1])
		if fromYear != "" {
			queryValues.Set("as_ylo", fromYear)
		}
		if toYear != "" {
			queryValues.Set("as_yhi", toYear)
		}
	}
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
