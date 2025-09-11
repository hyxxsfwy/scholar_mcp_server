package crossref

// Paper Crossref论文信息结构体
type Paper struct {
	DOI                 string    `json:"DOI"`                    // DOI
	URL                 string    `json:"URL"`                    // Crossref URL
	Title               []string  `json:"title"`                  // 标题
	Abstract            string    `json:"abstract"`               // 摘要
	ContainerTitle      []string  `json:"container-title"`        // 期刊/会议名称
	PublishedOnline     DateParts `json:"published-online"`       // 在线发表日期
	PublishedPrint      DateParts `json:"published-print"`        // 印刷发表日期
	Issued              DateParts `json:"issued"`                 // 发表日期
	Volume              string    `json:"volume"`                 // 卷号
	Issue               string    `json:"issue"`                  // 期号
	Page                string    `json:"page"`                   // 页码
	Publisher           string    `json:"publisher"`              // 出版商
	Type                string    `json:"type"`                   // 类型
	Subject             []string  `json:"subject"`                // 主题
	Author              []Author  `json:"author"`                 // 作者列表
	IsReferencedByCount int       `json:"is-referenced-by-count"` // 被引用次数
	ReferencesCount     int       `json:"references-count"`       // 引用次数
	License             []License `json:"license"`                // 许可证信息
	Funder              []Funder  `json:"funder"`                 // 资助信息
	Language            string    `json:"language"`               // 语言
	ShortTitle          []string  `json:"short-title"`            // 短标题
}

// Author 作者信息
type Author struct {
	Given       string        `json:"given"`       // 名
	Family      string        `json:"family"`      // 姓
	Sequence    string        `json:"sequence"`    // 顺序 (first, additional)
	Affiliation []Affiliation `json:"affiliation"` // 机构信息
}

// Affiliation 机构信息
type Affiliation struct {
	Name string `json:"name"` // 机构名称
}

// DateParts 日期结构
type DateParts struct {
	DateParts [][]int `json:"date-parts"` // 日期部分 [[年, 月, 日]]
}

// License 许可证信息
type License struct {
	URL            string    `json:"URL"`             // 许可证URL
	Start          DateParts `json:"start"`           // 开始日期
	DelayInDays    int       `json:"delay-in-days"`   // 延迟天数
	ContentVersion string    `json:"content-version"` // 内容版本
}

// Funder 资助信息
type Funder struct {
	DOI   string   `json:"DOI"`   // 资助机构DOI
	Name  string   `json:"name"`  // 资助机构名称
	Award []string `json:"award"` // 资助奖项
}

// SearchResult 搜索结果结构体
type SearchResult struct {
	Query        string  `json:"query"`          // 查询关键词
	Offset       int     `json:"offset"`         // 偏移量
	Rows         int     `json:"rows"`           // 行数
	TotalResults int     `json:"total-results"`  // 总结果数
	ItemsPerPage int     `json:"items-per-page"` // 每页项目数
	Papers       []Paper `json:"items"`          // 论文列表
}

// CrossrefSearchParam 搜索参数
type CrossrefSearchParam struct {
	Query      string `json:"query"`       // 搜索关键词
	Author     string `json:"author"`      // 作者筛选
	Title      string `json:"title"`       // 标题筛选
	Container  string `json:"container"`   // 期刊/会议筛选
	Publisher  string `json:"publisher"`   // 出版商筛选
	Type       string `json:"type"`        // 类型筛选 (journal-article, conference-paper, etc.)
	FromDate   string `json:"from_date"`   // 开始日期 (YYYY-MM-DD)
	UntilDate  string `json:"until_date"`  // 结束日期 (YYYY-MM-DD)
	HasOrcid   bool   `json:"has_orcid"`   // 是否有ORCID
	HasFunder  bool   `json:"has_funder"`  // 是否有资助信息
	HasLicense bool   `json:"has_license"` // 是否有许可证
	Offset     int    `json:"offset"`      // 偏移量，默认0
	Rows       int    `json:"rows"`        // 行数，默认20
	Sort       string `json:"sort"`        // 排序方式: relevance, published, updated
	Order      string `json:"order"`       // 排序顺序: asc, desc
}

// CrossrefGetParam 获取特定论文参数
type CrossrefGetParam struct {
	DOI string `json:"doi"` // DOI
}

// API响应结构体
type APIResponse struct {
	Status         string      `json:"status"`
	MessageType    string      `json:"message-type"`
	MessageVersion string      `json:"message-version"`
	Message        MessageData `json:"message"`
}

// MessageData 消息数据
type MessageData struct {
	Query        QueryInfo `json:"query"`
	ItemsPerPage int       `json:"items-per-page"`
	TotalResults int       `json:"total-results"`
	Items        []Paper   `json:"items"`
}

// QueryInfo 查询信息
type QueryInfo struct {
	StartIndex  int    `json:"start-index"`
	SearchTerms string `json:"search-terms"`
}
