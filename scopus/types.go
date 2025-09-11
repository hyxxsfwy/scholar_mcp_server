package scopus

// Paper Scopus论文信息结构体
type Paper struct {
	EID              string     `json:"eid"`              // Scopus EID
	DOI              string     `json:"doi"`              // DOI
	URL              string     `json:"url"`              // Scopus URL
	Title            string     `json:"title"`            // 标题
	Abstract         string     `json:"abstract"`         // 摘要
	PublicationName  string     `json:"publicationName"`  // 发表刊物名称
	Volume           string     `json:"volume"`           // 卷号
	IssueIdentifier  string     `json:"issueIdentifier"`  // 期号
	PageRange        string     `json:"pageRange"`        // 页码范围
	CoverDate        string     `json:"coverDate"`        // 发表日期
	Publisher        string     `json:"publisher"`        // 出版商
	ArticleNumber    string     `json:"articleNumber"`    // 文章编号
	AggregationType  string     `json:"aggregationType"`  // 聚合类型
	SubtypeDescription string   `json:"subtypeDescription"` // 子类型描述
	CitedByCount     int        `json:"citedByCount"`     // 被引用次数
	Author           []Author   `json:"author"`           // 作者列表
	Affiliation      []Affiliation `json:"affiliation"`   // 机构列表
	SubjectAreas     []SubjectArea `json:"subject-areas"`  // 主题领域
	Language         string     `json:"language"`         // 语言
	Fund             []Fund     `json:"fund-sponsor"`     // 资助信息
	OpenAccess       OpenAccess `json:"openaccess"`       // 开放获取信息
}

// Author 作者信息
type Author struct {
	AuthorID   string `json:"authid"`   // 作者ID
	AuthorName string `json:"authname"` // 作者姓名
	Surname    string `json:"surname"`  // 姓
	GivenName  string `json:"given-name"` // 名
	Initials   string `json:"initials"` // 缩写
	AffilID    string `json:"afid"`     // 机构ID
}

// Affiliation 机构信息
type Affiliation struct {
	AffilID     string `json:"afid"`     // 机构ID
	AffilName   string `json:"affilname"` // 机构名称
	City        string `json:"affiliation-city"` // 城市
	Country     string `json:"affiliation-country"` // 国家
}

// SubjectArea 主题领域
type SubjectArea struct {
	Code         string `json:"@code"`  // 代码
	Abbreviation string `json:"@abbrev"` // 缩写
	Name         string `json:"$"`      // 名称
}

// Fund 资助信息
type Fund struct {
	FundID     string `json:"@fund-id"`     // 资助ID
	FundAcronym string `json:"@fund-acronym"` // 资助机构缩写
	FundSponsor string `json:"fund-sponsor"` // 资助机构
}

// OpenAccess 开放获取信息
type OpenAccess struct {
	Flag           string `json:"@flag"`           // 标志 (true/false)
	UserLicense    string `json:"@user-license"`   // 用户许可证
	Type           string `json:"@type"`           // 类型
}

// SearchResult 搜索结果结构体
type SearchResult struct {
	Query       string  `json:"query"`       // 查询关键词
	Start       int     `json:"start"`       // 开始位置
	Count       int     `json:"count"`       // 数量
	TotalCount  int     `json:"totalCount"`  // 总数量
	Papers      []Paper `json:"entry"`       // 论文列表
}

// ScopusSearchParam 搜索参数
type ScopusSearchParam struct {
	Query       string   `json:"query"`       // 搜索关键词
	Author      string   `json:"author"`      // 作者筛选
	Title       string   `json:"title"`       // 标题筛选
	Journal     string   `json:"journal"`     // 期刊筛选
	Year        string   `json:"year"`        // 年份筛选 (例如: "2020-2023")
	SubjectArea string   `json:"subject_area"` // 主题领域筛选
	Affiliation string   `json:"affiliation"` // 机构筛选
	Funding     string   `json:"funding"`     // 资助机构筛选
	Language    string   `json:"language"`    // 语言筛选
	DocType     string   `json:"doc_type"`    // 文档类型筛选
	Start       int      `json:"start"`       // 开始位置，默认0
	Count       int      `json:"count"`       // 数量，默认25
	Sort        string   `json:"sort"`        // 排序方式: relevancy, pubyear, citedby-count, etc.
	View        string   `json:"view"`        // 视图类型: STANDARD, COMPLETE
}

// ScopusGetParam 获取特定论文参数
type ScopusGetParam struct {
	EID  string `json:"eid"`  // Scopus EID
	DOI  string `json:"doi"`  // DOI
	View string `json:"view"` // 视图类型: STANDARD, COMPLETE
}

// API响应结构体
type APIResponse struct {
	SearchResults SearchResults `json:"search-results"`
}

// SearchResults 搜索结果
type SearchResults struct {
	TotalResults string  `json:"opensearch:totalResults"`
	StartIndex   string  `json:"opensearch:startIndex"`
	ItemsPerPage string  `json:"opensearch:itemsPerPage"`
	Query        QueryInfo `json:"opensearch:Query"`
	Link         []Link  `json:"link"`
	Entry        []Paper `json:"entry"`
}

// QueryInfo 查询信息
type QueryInfo struct {
	Role        string `json:"@role"`
	SearchTerms string `json:"@searchTerms"`
	StartPage   string `json:"@startPage"`
}

// Link 链接信息
type Link struct {
	Ref  string `json:"@ref"`
	Href string `json:"@href"`
	Type string `json:"@type"`
}

// SinglePaperResponse 单篇论文响应
type SinglePaperResponse struct {
	AbstractRetrievalResponse AbstractRetrievalData `json:"abstracts-retrieval-response"`
}

// AbstractRetrievalData 论文摘要检索数据
type AbstractRetrievalData struct {
	CoreData CoreData `json:"coredata"`
	Authors  Authors  `json:"authors"`
	Affiliation []Affiliation `json:"affiliation"`
	SubjectAreas SubjectAreas `json:"subject-areas"`
	Item     Item     `json:"item"`
}

// CoreData 核心数据
type CoreData struct {
	EID              string `json:"eid"`
	DOI              string `json:"doi"`
	Title            string `json:"dc:title"`
	PublicationName  string `json:"prism:publicationName"`
	CoverDate        string `json:"prism:coverDate"`
	Creator          string `json:"dc:creator"`
	Description      string `json:"dc:description"`
	CitedByCount     string `json:"citedby-count"`
	Publisher        string `json:"dc:publisher"`
	AggregationType  string `json:"prism:aggregationType"`
	SubtypeDescription string `json:"subtypeDescription"`
	Link             []Link `json:"link"`
}

// Authors 作者信息
type Authors struct {
	Author []Author `json:"author"`
}

// SubjectAreas 主题领域
type SubjectAreas struct {
	SubjectArea []SubjectArea `json:"subject-area"`
}

// Item 项目信息
type Item struct {
	Bibrecord Bibrecord `json:"bibrecord"`
}

// Bibrecord 书目记录
type Bibrecord struct {
	Head Head `json:"head"`
	Tail Tail `json:"tail"`
}

// Head 头部信息
type Head struct {
	Abstract []Abstract `json:"abstracts"`
}

// Abstract 摘要信息
type Abstract struct {
	AbstractText string `json:"ce:abstract-text"`
}

// Tail 尾部信息
type Tail struct {
	Bibliography Bibliography `json:"bibliography"`
}

// Bibliography 参考文献
type Bibliography struct {
	Reference []Reference `json:"reference"`
}

// Reference 引用信息
type Reference struct {
	Position string `json:"@position"`
	ID       string `json:"@id"`
}
