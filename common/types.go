package common

import "context"

// UnifiedPaper 统一的论文信息结构体
type UnifiedPaper struct {
	// 基本信息
	ID       string   `json:"id"`       // 论文ID (各数据源的主要标识符)
	DOI      string   `json:"doi"`      // DOI
	Title    string   `json:"title"`    // 标题
	Abstract string   `json:"abstract"` // 摘要
	Authors  []string `json:"authors"`  // 作者列表

	// 发表信息
	Journal       string `json:"journal"`        // 期刊/会议名称
	Publisher     string `json:"publisher"`      // 出版商
	PublishedDate string `json:"published_date"` // 发表日期
	Volume        string `json:"volume"`         // 卷号
	Issue         string `json:"issue"`          // 期号
	Pages         string `json:"pages"`          // 页码

	// 分类和主题
	Categories []string `json:"categories"` // 分类/主题领域
	Keywords   []string `json:"keywords"`   // 关键词
	Language   string   `json:"language"`   // 语言

	// 引用和影响力
	CitationCount int `json:"citation_count"` // 被引用次数
	ReadCount     int `json:"read_count"`     // 阅读次数

	// 链接和获取
	URL          string `json:"url"`            // 原始链接
	PDFURL       string `json:"pdf_url"`        // PDF链接
	IsOpenAccess bool   `json:"is_open_access"` // 是否开放获取

	// 附加信息
	Type        string                 `json:"type"`         // 文档类型
	Source      string                 `json:"source"`       // 数据源
	ExternalIDs map[string]string      `json:"external_ids"` // 外部ID映射
	Metadata    map[string]interface{} `json:"metadata"`     // 额外元数据
}

// UnifiedSearchResult 统一的搜索结果结构体
type UnifiedSearchResult struct {
	Query       string         `json:"query"`       // 查询关键词
	Source      string         `json:"source"`      // 数据源
	Total       int            `json:"total"`       // 总数量
	Offset      int            `json:"offset"`      // 偏移量
	Limit       int            `json:"limit"`       // 限制数量
	Papers      []UnifiedPaper `json:"papers"`      // 论文列表
	SearchTime  int64          `json:"search_time"` // 搜索耗时(毫秒)
	Suggestions []string       `json:"suggestions"` // 搜索建议
}

// SearchParams 统一的搜索参数
type SearchParams struct {
	// 基本搜索参数
	Query  string `json:"query"`  // 搜索关键词
	Offset int    `json:"offset"` // 偏移量
	Limit  int    `json:"limit"`  // 限制数量

	// 过滤参数
	Author     string   `json:"author"`     // 作者筛选
	Title      string   `json:"title"`      // 标题筛选
	Abstract   string   `json:"abstract"`   // 摘要筛选
	Journal    string   `json:"journal"`    // 期刊筛选
	Publisher  string   `json:"publisher"`  // 出版商筛选
	Year       string   `json:"year"`       // 年份筛选
	YearRange  string   `json:"year_range"` // 年份范围
	Language   string   `json:"language"`   // 语言筛选
	Type       string   `json:"type"`       // 文档类型筛选
	Categories []string `json:"categories"` // 分类筛选

	// 排序参数
	SortBy    string `json:"sort_by"`    // 排序字段
	SortOrder string `json:"sort_order"` // 排序顺序

	// 高级参数
	OpenAccessOnly bool     `json:"open_access_only"` // 仅开放获取
	MinCitations   int      `json:"min_citations"`    // 最小引用数
	Fields         []string `json:"fields"`           // 返回字段
}

// PaperRetriever 论文检索接口
type PaperRetriever interface {
	// SearchPapers 搜索论文
	SearchPapers(ctx context.Context, params SearchParams) (*UnifiedSearchResult, error)

	// GetPaper 获取特定论文
	GetPaper(ctx context.Context, id string) (*UnifiedPaper, error)

	// GetSource 获取数据源名称
	GetSource() string

	// IsAvailable 检查服务是否可用
	IsAvailable(ctx context.Context) bool
}

// PDFRetriever PDF获取接口
type PDFRetriever interface {
	// GetPDFURL 获取PDF下载链接
	GetPDFURL(ctx context.Context, identifier string) (string, error)

	// CheckPDFAvailability 检查PDF是否可用
	CheckPDFAvailability(ctx context.Context, identifier string) (bool, error)
}

// CitationRetriever 引用信息接口
type CitationRetriever interface {
	// GetCitations 获取引用论文列表
	GetCitations(ctx context.Context, paperID string, offset, limit int) ([]UnifiedPaper, error)

	// GetReferences 获取参考文献列表
	GetReferences(ctx context.Context, paperID string, offset, limit int) ([]UnifiedPaper, error)

	// GetCitationCount 获取引用数量
	GetCitationCount(ctx context.Context, paperID string) (int, error)
}

// MetricsRetriever 指标信息接口
type MetricsRetriever interface {
	// GetMetrics 获取论文指标
	GetMetrics(ctx context.Context, paperID string) (*PaperMetrics, error)
}

// ExportRetriever 导出功能接口
type ExportRetriever interface {
	// ExportCitation 导出引用格式
	ExportCitation(ctx context.Context, paperIDs []string, format string) (string, error)

	// GetSupportedFormats 获取支持的导出格式
	GetSupportedFormats() []string
}

// PaperMetrics 论文指标
type PaperMetrics struct {
	PaperID              string  `json:"paper_id"`
	CitationCount        int     `json:"citation_count"`
	ReadCount            int     `json:"read_count"`
	DownloadCount        int     `json:"download_count"`
	HIndex               int     `json:"h_index"`
	ImpactFactor         float64 `json:"impact_factor"`
	InfluentialCitations int     `json:"influential_citations"`
	RecentCitations      int     `json:"recent_citations"`
	SelfCitations        int     `json:"self_citations"`

	// 时间序列数据
	CitationsTimeSeries []TimeSeriesPoint `json:"citations_time_series"`
	ReadsTimeSeries     []TimeSeriesPoint `json:"reads_time_series"`
}

// TimeSeriesPoint 时间序列数据点
type TimeSeriesPoint struct {
	Date  string `json:"date"`  // 日期
	Value int    `json:"value"` // 数值
}

// SearchHint 搜索提示
type SearchHint struct {
	Query       string  `json:"query"`       // 建议查询
	Type        string  `json:"type"`        // 提示类型
	Confidence  float64 `json:"confidence"`  // 置信度
	Description string  `json:"description"` // 描述
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Code    string `json:"code"`    // 错误代码
	Message string `json:"message"` // 错误消息
	Source  string `json:"source"`  // 错误来源
	Details string `json:"details"` // 详细信息
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Source       string `json:"source"`        // 数据源
	Available    bool   `json:"available"`     // 是否可用
	ResponseTime int64  `json:"response_time"` // 响应时间(毫秒)
	LastCheck    string `json:"last_check"`    // 最后检查时间
	Message      string `json:"message"`       // 状态消息
}

// AggregatedSearchResult 聚合搜索结果
type AggregatedSearchResult struct {
	Query         string                `json:"query"`          // 查询关键词
	TotalSources  int                   `json:"total_sources"`  // 总数据源数
	ActiveSources int                   `json:"active_sources"` // 活跃数据源数
	TotalResults  int                   `json:"total_results"`  // 总结果数
	SearchTime    int64                 `json:"search_time"`    // 搜索耗时
	Results       []UnifiedSearchResult `json:"results"`        // 各源结果
	MergedPapers  []UnifiedPaper        `json:"merged_papers"`  // 合并去重后的论文
	SourceStatus  []ServiceStatus       `json:"source_status"`  // 各源状态
	Suggestions   []SearchHint          `json:"suggestions"`    // 搜索建议
}

// Config 配置信息
type Config struct {
	// API密钥配置
	SemanticScholarAPIKey string `json:"semantic_scholar_api_key"`
	ScopusAPIKey          string `json:"scopus_api_key"`
	AdsabsAPIKey          string `json:"adsabs_api_key"`
	GoogleScholarAPIKey   string `json:"google_scholar_api_key"`

	// 请求配置
	RequestTimeout  int  `json:"request_timeout"`  // 请求超时时间(秒)
	MaxRetries      int  `json:"max_retries"`      // 最大重试次数
	EnableCache     bool `json:"enable_cache"`     // 是否启用缓存
	CacheExpiration int  `json:"cache_expiration"` // 缓存过期时间(秒)

	// 搜索配置
	DefaultLimit int    `json:"default_limit"` // 默认搜索限制
	MaxLimit     int    `json:"max_limit"`     // 最大搜索限制
	DefaultSort  string `json:"default_sort"`  // 默认排序

	// 功能开关
	EnableSemanticScholar bool `json:"enable_semantic_scholar"`
	EnableCrossref        bool `json:"enable_crossref"`
	EnableScopus          bool `json:"enable_scopus"`
	EnableAdsabs          bool `json:"enable_adsabs"`
	EnableGoogleScholar   bool `json:"enable_google_scholar"`
	EnableSciHub          bool `json:"enable_scihub"`
	EnableArxiv           bool `json:"enable_arxiv"`
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field"`   // 字段名
	Message string `json:"message"` // 错误消息
	Value   string `json:"value"`   // 错误值
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid  bool              `json:"valid"`  // 是否有效
	Errors []ValidationError `json:"errors"` // 错误列表
}
