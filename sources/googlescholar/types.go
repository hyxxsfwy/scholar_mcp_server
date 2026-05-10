package googlescholar

// Paper represents one Google Scholar organic result returned by a SerpAPI-compatible API.
type Paper struct {
	Position        int             `json:"position"`
	ResultID        string          `json:"result_id"`
	Title           string          `json:"title"`
	Link            string          `json:"link"`
	Snippet         string          `json:"snippet"`
	PublicationInfo PublicationInfo `json:"publication_info"`
	InlineLinks     InlineLinks     `json:"inline_links"`
	Resources       []Resource      `json:"resources"`
}

type PublicationInfo struct {
	Summary string   `json:"summary"`
	Authors []Author `json:"authors"`
}

type Author struct {
	Name               string `json:"name"`
	Link               string `json:"link"`
	SerpAPIScholarLink string `json:"serpapi_scholar_link"`
	AuthorID           string `json:"author_id"`
}

type InlineLinks struct {
	CitedBy          CitedBy  `json:"cited_by"`
	Versions         Versions `json:"versions"`
	RelatedPagesLink string   `json:"related_pages_link"`
}

type CitedBy struct {
	Total              int    `json:"total"`
	Link               string `json:"link"`
	SerpAPIScholarLink string `json:"serpapi_scholar_link"`
	CitesID            string `json:"cites_id"`
}

type Versions struct {
	Total              int    `json:"total"`
	Link               string `json:"link"`
	SerpAPIScholarLink string `json:"serpapi_scholar_link"`
	ClusterID          string `json:"cluster_id"`
}

type Resource struct {
	Title      string `json:"title"`
	FileFormat string `json:"file_format"`
	Link       string `json:"link"`
}

type SearchResult struct {
	Query  string  `json:"query"`
	Offset int     `json:"offset"`
	Limit  int     `json:"limit"`
	Total  int     `json:"total"`
	Papers []Paper `json:"organic_results"`
}

type GoogleScholarSearchParam struct {
	Query     string `json:"query"`
	Author    string `json:"author"`
	Journal   string `json:"journal"`
	Year      string `json:"year"`
	YearRange string `json:"year_range"`
	Offset    int    `json:"offset"`
	Limit     int    `json:"limit"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"`
}

type APIResponse struct {
	SearchInformation SearchInformation `json:"search_information"`
	OrganicResults    []Paper           `json:"organic_results"`
	Error             string            `json:"error"`
}

type SearchInformation struct {
	TotalResults int `json:"total_results"`
}
