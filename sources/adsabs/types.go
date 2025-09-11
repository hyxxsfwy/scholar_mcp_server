package adsabs

// Paper ADSABS论文信息结构体
type Paper struct {
	Bibcode       string   `json:"bibcode"`        // ADSABS Bibcode
	Identifier    []string `json:"identifier"`     // 标识符列表
	Title         []string `json:"title"`          // 标题
	Abstract      string   `json:"abstract"`       // 摘要
	Author        []string `json:"author"`         // 作者列表
	FirstAuthor   string   `json:"first_author"`   // 第一作者
	Pub           string   `json:"pub"`            // 期刊名称
	PubDate       string   `json:"pubdate"`        // 发表日期
	Volume        string   `json:"volume"`         // 卷号
	Issue         string   `json:"issue"`          // 期号
	Page          []string `json:"page"`           // 页码
	DOI           []string `json:"doi"`            // DOI列表
	ArxivID       []string `json:"arxiv_id"`       // arXiv ID列表
	Database      []string `json:"database"`       // 数据库
	DocType       string   `json:"doctype"`        // 文档类型
	CitationCount int      `json:"citation_count"` // 被引用次数
	ReadCount     int      `json:"read_count"`     // 阅读次数
	Property      []string `json:"property"`       // 属性
	Keyword       []string `json:"keyword"`        // 关键词
	Bibstem       []string `json:"bibstem"`        // 期刊缩写
	Aff           []string `json:"aff"`            // 机构信息
	Facility      []string `json:"facility"`       // 设施
	Instrument    []string `json:"instrument"`     // 仪器
	Object        []string `json:"object"`         // 天体对象
	ORCID         []string `json:"orcid"`          // ORCID ID
	Grant         []string `json:"grant"`          // 资助信息
}

// SearchResult 搜索结果结构体
type SearchResult struct {
	Query          string         `json:"query"`          // 查询关键词
	Start          int            `json:"start"`          // 开始位置
	Rows           int            `json:"rows"`           // 行数
	NumFound       int            `json:"numFound"`       // 找到的数量
	Papers         []Paper        `json:"docs"`           // 论文列表
	ResponseHeader ResponseHeader `json:"responseHeader"` // 响应头
}

// ResponseHeader 响应头信息
type ResponseHeader struct {
	Status int               `json:"status"` // 状态码
	QTime  int               `json:"QTime"`  // 查询时间
	Params map[string]string `json:"params"` // 参数
}

// AdsabsSearchParam 搜索参数
type AdsabsSearchParam struct {
	Query    string   `json:"query"`    // 搜索关键词
	Author   string   `json:"author"`   // 作者筛选
	Title    string   `json:"title"`    // 标题筛选
	Abstract string   `json:"abstract"` // 摘要筛选
	Year     string   `json:"year"`     // 年份筛选 (例如: "2020-2023")
	Pub      string   `json:"pub"`      // 期刊筛选
	Object   string   `json:"object"`   // 天体对象筛选
	Database string   `json:"database"` // 数据库筛选 (astronomy, physics)
	Property string   `json:"property"` // 属性筛选 (refereed, notrefereed, etc.)
	DocType  string   `json:"doctype"`  // 文档类型筛选
	Bibstem  string   `json:"bibstem"`  // 期刊缩写筛选
	Start    int      `json:"start"`    // 开始位置，默认0
	Rows     int      `json:"rows"`     // 行数，默认10
	Sort     string   `json:"sort"`     // 排序方式: date desc, citation_count desc, read_count desc
	Fields   []string `json:"fields"`   // 返回字段列表
}

// AdsabsGetParam 获取特定论文参数
type AdsabsGetParam struct {
	Bibcode string   `json:"bibcode"` // ADSABS Bibcode
	Fields  []string `json:"fields"`  // 返回字段列表
}

// API响应结构体
type APIResponse struct {
	ResponseHeader ResponseHeader `json:"responseHeader"`
	Response       ResponseData   `json:"response"`
}

// ResponseData 响应数据
type ResponseData struct {
	NumFound int     `json:"numFound"`
	Start    int     `json:"start"`
	Docs     []Paper `json:"docs"`
}

// Export格式选项
type ExportFormat string

const (
	ExportBibTeX       ExportFormat = "bibtex"
	ExportEndNote      ExportFormat = "endnote"
	ExportRIS          ExportFormat = "ris"
	ExportAASTex       ExportFormat = "aastex"
	ExportIcarus       ExportFormat = "icarus"
	ExportMNRAS        ExportFormat = "mnras"
	ExportSolarPhysics ExportFormat = "soph"
)

// ExportRequest 导出请求
type ExportRequest struct {
	Bibcode []string     `json:"bibcode"` // Bibcode列表
	Format  ExportFormat `json:"format"`  // 导出格式
	Sort    string       `json:"sort"`    // 排序方式
}

// ExportResponse 导出响应
type ExportResponse struct {
	Export string `json:"export"` // 导出内容
	Msg    string `json:"msg"`    // 消息
}

// MetricsResponse 指标响应
type MetricsResponse struct {
	Bibcode               string                 `json:"bibcode"`
	CitationCount         int                    `json:"citation_count"`
	ReadCount             int                    `json:"read_count"`
	Downloads             []HistogramData        `json:"downloads"`
	Reads                 []HistogramData        `json:"reads"`
	Citations             []HistogramData        `json:"citations"`
	CitationHIndex        int                    `json:"citation_h_index"`
	CitationGIndex        int                    `json:"citation_g_index"`
	CitationM             float64                `json:"citation_m"`
	CitationI10           int                    `json:"citation_i10"`
	CitationTori          float64                `json:"citation_tori"`
	CitationRiq           float64                `json:"citation_riq"`
	RefereedCitationCount int                    `json:"refereed_citation_count"`
	TotalCitationCount    int                    `json:"total_citation_count"`
	TotalReadCount        int                    `json:"total_read_count"`
	HIndex                int                    `json:"h_index"`
	GIndex                int                    `json:"g_index"`
	MIndex                float64                `json:"m_index"`
	I10Index              int                    `json:"i10_index"`
	ToriIndex             float64                `json:"tori_index"`
	RiqIndex              float64                `json:"riq_index"`
	TimeSeries            map[string]interface{} `json:"time_series"`
}

// HistogramData 历史数据
type HistogramData struct {
	Year  string `json:"year"`
	Count int    `json:"count"`
}
