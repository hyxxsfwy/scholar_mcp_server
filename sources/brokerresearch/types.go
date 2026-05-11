package brokerresearch

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	DefaultLimit = 10
	MaxLimit     = 50
)

type FeedConfig struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`
	Format  string            `json:"format"`
	Broker  string            `json:"broker"`
	Team    string            `json:"team"`
	APIKey  string            `json:"api_key"`
	Headers map[string]string `json:"headers"`
}

type SearchParam struct {
	Query          string   `json:"query"`
	Author         string   `json:"author"`
	Title          string   `json:"title"`
	Abstract       string   `json:"abstract"`
	Journal        string   `json:"journal"`
	Publisher      string   `json:"publisher"`
	Year           string   `json:"year"`
	YearRange      string   `json:"year_range"`
	Language       string   `json:"language"`
	Type           string   `json:"type"`
	Categories     []string `json:"categories"`
	OpenAccessOnly bool     `json:"open_access_only"`
	Offset         int      `json:"offset"`
	Limit          int      `json:"limit"`
	SortBy         string   `json:"sort_by"`
	SortOrder      string   `json:"sort_order"`
}

type SearchResult struct {
	Query   string   `json:"query"`
	Offset  int      `json:"offset"`
	Limit   int      `json:"limit"`
	Total   int      `json:"total"`
	Reports []Report `json:"reports"`
}

type Report struct {
	ID            string                 `json:"id"`
	GUID          string                 `json:"guid"`
	DOI           string                 `json:"doi"`
	Title         string                 `json:"title"`
	Abstract      string                 `json:"abstract"`
	Summary       string                 `json:"summary"`
	Authors       StringList             `json:"authors"`
	Author        string                 `json:"author"`
	Broker        string                 `json:"broker"`
	Team          string                 `json:"team"`
	Publisher     string                 `json:"publisher"`
	SourceName    string                 `json:"source"`
	PublishedDate string                 `json:"published_date"`
	PublishedAt   string                 `json:"published_at"`
	PublishDate   string                 `json:"publish_date"`
	Date          string                 `json:"date"`
	URL           string                 `json:"url"`
	Link          string                 `json:"link"`
	PDFURL        string                 `json:"pdf_url"`
	PDF           string                 `json:"pdf"`
	Categories    StringList             `json:"categories"`
	Keywords      StringList             `json:"keywords"`
	Tags          StringList             `json:"tags"`
	Language      string                 `json:"language"`
	Type          string                 `json:"type"`
	ReadCount     FlexibleInt            `json:"read_count"`
	Score         FlexibleInt            `json:"score"`
	Metadata      map[string]interface{} `json:"metadata"`
	Raw           map[string]interface{} `json:"-"`
}

type StringList []string

func (list *StringList) UnmarshalJSON(data []byte) error {
	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	*list = normalizeStringList(value)
	return nil
}

type FlexibleInt int

func (value *FlexibleInt) UnmarshalJSON(data []byte) error {
	var number int
	if err := json.Unmarshal(data, &number); err == nil {
		*value = FlexibleInt(number)
		return nil
	}

	var numberString string
	if err := json.Unmarshal(data, &numberString); err == nil {
		parsed, parseErr := strconv.Atoi(strings.TrimSpace(numberString))
		if parseErr != nil {
			*value = 0
			return nil
		}
		*value = FlexibleInt(parsed)
		return nil
	}

	*value = 0
	return nil
}

func normalizeStringList(value interface{}) []string {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return splitListText(typed)
	case []interface{}:
		return normalizeMixedStringList(typed)
	default:
		return nil
	}
}

func normalizeMixedStringList(values []interface{}) []string {
	var result []string
	for _, item := range values {
		result = append(result, stringListItemValues(item)...)
	}
	return result
}

func stringListItemValues(item interface{}) []string {
	switch itemValue := item.(type) {
	case string:
		return splitListText(itemValue)
	case map[string]interface{}:
		return splitListText(firstStringListField(itemValue))
	default:
		return nil
	}
}

func firstStringListField(item map[string]interface{}) string {
	for _, key := range []string{"name", "title", "value", "label"} {
		if text, ok := item[key].(string); ok {
			return text
		}
	}
	return ""
}

func splitListText(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	replacer := strings.NewReplacer("；", ";", "，", ",", "、", ",", "|", ",")
	text = replacer.Replace(text)
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == ';'
	})

	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Join(strings.Fields(strings.TrimSpace(part)), " ")
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}

	if len(cleaned) == 0 {
		return []string{text}
	}
	return cleaned
}
