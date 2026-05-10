package openalex

type Paper struct {
	ID                    string            `json:"id"`
	DOI                   string            `json:"doi"`
	Title                 string            `json:"title"`
	DisplayName           string            `json:"display_name"`
	PublicationYear       int               `json:"publication_year"`
	PublicationDate       string            `json:"publication_date"`
	Type                  string            `json:"type"`
	TypeCrossref          string            `json:"type_crossref"`
	CitedByCount          int               `json:"cited_by_count"`
	IDs                   map[string]string `json:"ids"`
	Authorships           []Authorship      `json:"authorships"`
	PrimaryLocation       Location          `json:"primary_location"`
	BestOALocation        *Location         `json:"best_oa_location"`
	Locations             []Location        `json:"locations"`
	OpenAccess            OpenAccess        `json:"open_access"`
	Biblio                Biblio            `json:"biblio"`
	Concepts              []Concept         `json:"concepts"`
	Keywords              []Keyword         `json:"keywords"`
	AbstractInvertedIndex map[string][]int  `json:"abstract_inverted_index"`
}

type Authorship struct {
	Author       Author        `json:"author"`
	Institutions []Institution `json:"institutions"`
}

type Author struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Institution struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Location struct {
	IsOA           bool    `json:"is_oa"`
	LandingPageURL string  `json:"landing_page_url"`
	PDFURL         string  `json:"pdf_url"`
	Source         *Source `json:"source"`
}

type Source struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
	ISSNL       string `json:"issn_l"`
}

type OpenAccess struct {
	IsOA     bool   `json:"is_oa"`
	OAStatus string `json:"oa_status"`
	OAURL    string `json:"oa_url"`
}

type Biblio struct {
	Volume    string `json:"volume"`
	Issue     string `json:"issue"`
	FirstPage string `json:"first_page"`
	LastPage  string `json:"last_page"`
}

type Concept struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Score       float64 `json:"score"`
}

type Keyword struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Score       float64 `json:"score"`
}

type SearchResult struct {
	Query  string  `json:"query"`
	Offset int     `json:"offset"`
	Limit  int     `json:"limit"`
	Total  int     `json:"total"`
	Papers []Paper `json:"results"`
}

type OpenAlexSearchParam struct {
	Query          string `json:"query"`
	Year           string `json:"year"`
	YearRange      string `json:"year_range"`
	Type           string `json:"type"`
	OpenAccessOnly bool   `json:"open_access_only"`
	MinCitations   int    `json:"min_citations"`
	Offset         int    `json:"offset"`
	Limit          int    `json:"limit"`
	SortBy         string `json:"sort_by"`
	SortOrder      string `json:"sort_order"`
}

type APIResponse struct {
	Meta    Meta    `json:"meta"`
	Results []Paper `json:"results"`
	Error   string  `json:"error"`
}

type Meta struct {
	Count   int `json:"count"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}
