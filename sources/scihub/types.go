package scihub

// Paper Sci-Hub论文信息结构体
type Paper struct {
	DOI       string `json:"doi"`        // DOI
	Title     string `json:"title"`      // 标题
	Authors   string `json:"authors"`    // 作者
	Year      string `json:"year"`       // 年份
	Journal   string `json:"journal"`    // 期刊
	PDFURL    string `json:"pdf_url"`    // PDF下载链接
	Available bool   `json:"available"`  // 是否可用
	SciHubURL string `json:"scihub_url"` // Sci-Hub链接
	FileSize  string `json:"file_size"`  // 文件大小
	Citation  string `json:"citation"`   // 引用格式
}

// SearchResult 搜索结果结构体
type SearchResult struct {
	Query  string  `json:"query"`  // 查询关键词
	Papers []Paper `json:"papers"` // 论文列表
	Total  int     `json:"total"`  // 总数量
	Source string  `json:"source"` // 数据源
}

// SciHubSearchParam 搜索参数
type SciHubSearchParam struct {
	Query      string `json:"query"`       // 搜索关键词 (DOI, PMID, 标题等)
	MaxResults int    `json:"max_results"` // 最大结果数，默认10
}

// SciHubGetParam 获取特定论文参数
type SciHubGetParam struct {
	Identifier string `json:"identifier"` // 论文标识符 (DOI, PMID, URL)
}

// PDFResponse PDF响应
type PDFResponse struct {
	Success   bool   `json:"success"`    // 是否成功
	PDFURL    string `json:"pdf_url"`    // PDF URL
	SciHubURL string `json:"scihub_url"` // Sci-Hub URL
	Message   string `json:"message"`    // 消息
	FileSize  string `json:"file_size"`  // 文件大小
}

// Mirror Sci-Hub镜像信息
type Mirror struct {
	URL       string `json:"url"`        // 镜像URL
	Status    string `json:"status"`     // 状态 (active, inactive, unknown)
	Location  string `json:"location"`   // 地理位置
	LastCheck string `json:"last_check"` // 最后检查时间
}

// MirrorStatus 镜像状态响应
type MirrorStatus struct {
	Mirrors     []Mirror `json:"mirrors"`      // 镜像列表
	ActiveCount int      `json:"active_count"` // 活跃镜像数量
	LastUpdate  string   `json:"last_update"`  // 最后更新时间
}

// AvailabilityCheck 可用性检查
type AvailabilityCheck struct {
	DOI       string `json:"doi"`        // DOI
	Available bool   `json:"available"`  // 是否可用
	URL       string `json:"url"`        // 可用URL
	CheckTime string `json:"check_time"` // 检查时间
}

// BulkCheck 批量检查请求
type BulkCheckRequest struct {
	DOIs []string `json:"dois"` // DOI列表
}

// BulkCheckResponse 批量检查响应
type BulkCheckResponse struct {
	Results []AvailabilityCheck `json:"results"` // 检查结果列表
	Total   int                 `json:"total"`   // 总数
	Success int                 `json:"success"` // 成功数量
}

// Statistics 统计信息
type Statistics struct {
	TotalPapers     int64         `json:"total_papers"`     // 总论文数
	TotalDownloads  int64         `json:"total_downloads"`  // 总下载数
	Coverage        float64       `json:"coverage"`         // 覆盖率
	LastUpdate      string        `json:"last_update"`      // 最后更新
	PopularJournals []JournalStat `json:"popular_journals"` // 热门期刊
	RecentPapers    []Paper       `json:"recent_papers"`    // 最近论文
}

// JournalStat 期刊统计
type JournalStat struct {
	Name  string `json:"name"`  // 期刊名称
	Count int    `json:"count"` // 论文数量
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`   // 错误信息
	Code    int    `json:"code"`    // 错误代码
	Message string `json:"message"` // 详细消息
}

// SearchHint 搜索提示
type SearchHint struct {
	Type       string  `json:"type"`       // 提示类型 (doi, pmid, title, etc.)
	Suggestion string  `json:"suggestion"` // 建议
	Confidence float64 `json:"confidence"` // 置信度
}

// DownloadMetadata 下载元数据
type DownloadMetadata struct {
	DOI           string            `json:"doi"`            // DOI
	Title         string            `json:"title"`          // 标题
	Authors       string            `json:"authors"`        // 作者
	Journal       string            `json:"journal"`        // 期刊
	Year          string            `json:"year"`           // 年份
	Volume        string            `json:"volume"`         // 卷号
	Issue         string            `json:"issue"`          // 期号
	Pages         string            `json:"pages"`          // 页码
	Publisher     string            `json:"publisher"`      // 出版商
	ISSN          string            `json:"issn"`           // ISSN
	Language      string            `json:"language"`       // 语言
	Abstract      string            `json:"abstract"`       // 摘要
	Keywords      []string          `json:"keywords"`       // 关键词
	FileInfo      FileInfo          `json:"file_info"`      // 文件信息
	DownloadInfo  DownloadInfo      `json:"download_info"`  // 下载信息
	ExternalLinks map[string]string `json:"external_links"` // 外部链接
}

// FileInfo 文件信息
type FileInfo struct {
	Filename    string `json:"filename"`     // 文件名
	Size        string `json:"size"`         // 文件大小
	Format      string `json:"format"`       // 文件格式
	Quality     string `json:"quality"`      // 质量
	Pages       int    `json:"pages"`        // 页数
	CreatedDate string `json:"created_date"` // 创建日期
	MD5         string `json:"md5"`          // MD5校验值
}

// DownloadInfo 下载信息
type DownloadInfo struct {
	DirectURL  string            `json:"direct_url"`  // 直接下载链接
	MirrorURLs []string          `json:"mirror_urls"` // 镜像链接
	Headers    map[string]string `json:"headers"`     // 请求头
	Cookies    map[string]string `json:"cookies"`     // Cookie
	UserAgent  string            `json:"user_agent"`  // User-Agent
	Referer    string            `json:"referer"`     // Referer
	ExpiresAt  string            `json:"expires_at"`  // 过期时间
}
