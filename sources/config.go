package sources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const DefaultProjectConfigFile = "config.jsonc"
const DefaultDotEnvFile = ".env"

var (
	dotEnvLoadOnce sync.Once
	dotEnvLoadErr  error
)

type ProjectConfig struct {
	Server     ServerConfig                 `json:"server"`
	Request    RequestConfig                `json:"request"`
	Sources    map[string]SourceConfigPatch `json:"sources"`
	Enrichment EnrichmentConfigPatch        `json:"enrichment"`
}

type ServerConfig struct {
	HTTP *bool   `json:"http"`
	Port *string `json:"port"`
}

type RequestConfig struct {
	Timeout        *int `json:"timeout"`
	RequestTimeout *int `json:"request_timeout"`
	MaxRetries     *int `json:"max_retries"`
	RateLimit      *int `json:"rate_limit"`
}

type SourceConfigPatch struct {
	Enabled        *bool           `json:"enabled"`
	APIKey         *string         `json:"api_key"`
	Email          *string         `json:"email"`
	BaseURL        *string         `json:"base_url"`
	Feeds          json.RawMessage `json:"feeds"`
	Timeout        *int            `json:"timeout"`
	RequestTimeout *int            `json:"request_timeout"`
	MaxRetries     *int            `json:"max_retries"`
	RateLimit      *int            `json:"rate_limit"`
}

type ResultEnrichmentConfig struct {
	Enabled         bool
	TopN            int
	PrefetchPDF     bool
	PDFFallback     string
	MaxPDFBytes     int64
	PDFTextMaxChars int
	MaxInputChars   int
	TimeoutSeconds  int
	LLM             LLMConfig
}

type LLMConfig struct {
	APIKey         string
	BaseURL        string
	Model          string
	MaxTokens      int
	Temperature    float64
	MaxConcurrency int
}

type EnrichmentConfigPatch struct {
	Enabled         *bool          `json:"enabled"`
	TopN            *int           `json:"top_n"`
	PrefetchPDF     *bool          `json:"prefetch_pdf"`
	PDFFallback     *string        `json:"pdf_fallback"`
	MaxPDFBytes     *int64         `json:"max_pdf_bytes"`
	PDFTextMaxChars *int           `json:"pdf_text_max_chars"`
	MaxInputChars   *int           `json:"max_input_chars"`
	Timeout         *int           `json:"timeout"`
	TimeoutSeconds  *int           `json:"timeout_seconds"`
	LLM             LLMConfigPatch `json:"llm"`
}

type LLMConfigPatch struct {
	APIKey         *string  `json:"api_key"`
	BaseURL        *string  `json:"base_url"`
	Model          *string  `json:"model"`
	MaxTokens      *int     `json:"max_tokens"`
	Temperature    *float64 `json:"temperature"`
	MaxConcurrency *int     `json:"max_concurrency"`
}

func LoadProjectConfig() (*ProjectConfig, string, error) {
	path, ok, err := findProjectConfigFile(DefaultProjectConfigFile)
	if err != nil || !ok {
		return nil, "", err
	}

	config, err := LoadProjectConfigFromFile(path)
	if err != nil {
		return nil, path, err
	}

	return config, path, nil
}

func LoadProjectConfigFromFile(path string) (*ProjectConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseProjectConfigJSONC(content)
}

func ParseProjectConfigJSONC(content []byte) (*ProjectConfig, error) {
	jsonContent, err := jsoncToJSON(content)
	if err != nil {
		return nil, err
	}

	var config ProjectConfig
	decoder := json.NewDecoder(bytes.NewReader(jsonContent))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("解析JSONC配置失败: %w", err)
	}

	return &config, nil
}

// LoadDotEnv loads the nearest .env file into the process environment without overriding existing variables.
func LoadDotEnv() error {
	dotEnvLoadOnce.Do(func() {
		dotEnvLoadErr = loadDotEnvFromFile(DefaultDotEnvFile)
	})
	return dotEnvLoadErr
}

func (config *ProjectConfig) ApplyToSourceConfigs(configs map[string]SourceConfig) error {
	if config == nil {
		return nil
	}

	if requestPatch := config.Request.sourcePatch(); requestPatch != nil {
		for name, sourceConfig := range configs {
			configs[name] = requestPatch.apply(sourceConfig)
		}
	}

	for name, patch := range config.Sources {
		sourceConfig := configs[name]
		patched, err := patch.applyWithError(sourceConfig)
		if err != nil {
			return fmt.Errorf("应用数据源 %s 配置失败: %w", name, err)
		}
		configs[name] = patched
	}

	return nil
}

func (config ServerConfig) HTTPDefault(defaultValue bool) bool {
	if config.HTTP == nil {
		return defaultValue
	}
	return *config.HTTP
}

func (config ServerConfig) PortDefault(defaultValue string) string {
	if config.Port == nil || strings.TrimSpace(*config.Port) == "" {
		return defaultValue
	}
	return strings.TrimSpace(*config.Port)
}

func LoadResultEnrichmentConfig() ResultEnrichmentConfig {
	if err := LoadDotEnv(); err != nil {
		log.Printf("[WARN] 读取.env文件失败: %v", err)
	}

	config := defaultResultEnrichmentConfigFromEnv()
	projectConfig, _, err := LoadProjectConfig()
	if err == nil && projectConfig != nil {
		projectConfig.Enrichment.apply(&config)
	}

	normalizeResultEnrichmentConfig(&config)
	return config
}

func defaultResultEnrichmentConfigFromEnv() ResultEnrichmentConfig {
	return ResultEnrichmentConfig{
		Enabled:         boolEnv("ENABLE_RESULT_ENRICHMENT", false),
		TopN:            intEnv("RESULT_ENRICHMENT_TOP_N", 5),
		PrefetchPDF:     boolEnv("RESULT_ENRICHMENT_PREFETCH_PDF", true),
		PDFFallback:     firstEnvValue("RESULT_ENRICHMENT_PDF_FALLBACK", "PDF_FALLBACK"),
		MaxPDFBytes:     int64Env("RESULT_ENRICHMENT_MAX_PDF_BYTES", 10*1024*1024),
		PDFTextMaxChars: intEnv("RESULT_ENRICHMENT_PDF_TEXT_MAX_CHARS", 12000),
		MaxInputChars:   intEnv("RESULT_ENRICHMENT_MAX_INPUT_CHARS", 16000),
		TimeoutSeconds:  intEnv("RESULT_ENRICHMENT_TIMEOUT", 60),
		LLM: LLMConfig{
			APIKey:         firstEnvValue("RESULT_ENRICHMENT_LLM_API_KEY", "LLM_API_KEY", "DEEPSEEK_API_KEY"),
			BaseURL:        firstEnvValue("RESULT_ENRICHMENT_LLM_BASE_URL", "LLM_BASE_URL", "DEEPSEEK_BASE_URL"),
			Model:          firstEnvValue("RESULT_ENRICHMENT_LLM_MODEL", "LLM_MODEL", "DEEPSEEK_MODEL"),
			MaxTokens:      intEnv("RESULT_ENRICHMENT_LLM_MAX_TOKENS", 1200),
			Temperature:    floatEnv("RESULT_ENRICHMENT_LLM_TEMPERATURE", 0.2),
			MaxConcurrency: intEnv("RESULT_ENRICHMENT_LLM_MAX_CONCURRENCY", 5),
		},
	}
}

func (patch EnrichmentConfigPatch) apply(config *ResultEnrichmentConfig) {
	if patch.Enabled != nil {
		config.Enabled = *patch.Enabled
	}
	if patch.TopN != nil {
		config.TopN = *patch.TopN
	}
	if patch.PrefetchPDF != nil {
		config.PrefetchPDF = *patch.PrefetchPDF
	}
	if patch.PDFFallback != nil {
		config.PDFFallback = *patch.PDFFallback
	}
	if patch.MaxPDFBytes != nil {
		config.MaxPDFBytes = *patch.MaxPDFBytes
	}
	if patch.PDFTextMaxChars != nil {
		config.PDFTextMaxChars = *patch.PDFTextMaxChars
	}
	if patch.MaxInputChars != nil {
		config.MaxInputChars = *patch.MaxInputChars
	}
	if patch.Timeout != nil {
		config.TimeoutSeconds = *patch.Timeout
	}
	if patch.TimeoutSeconds != nil {
		config.TimeoutSeconds = *patch.TimeoutSeconds
	}
	patch.LLM.apply(&config.LLM)
}

func (patch LLMConfigPatch) apply(config *LLMConfig) {
	if patch.APIKey != nil {
		config.APIKey = *patch.APIKey
	}
	if patch.BaseURL != nil {
		config.BaseURL = *patch.BaseURL
	}
	if patch.Model != nil {
		config.Model = *patch.Model
	}
	if patch.MaxTokens != nil {
		config.MaxTokens = *patch.MaxTokens
	}
	if patch.Temperature != nil {
		config.Temperature = *patch.Temperature
	}
	if patch.MaxConcurrency != nil {
		config.MaxConcurrency = *patch.MaxConcurrency
	}
}

func normalizeResultEnrichmentConfig(config *ResultEnrichmentConfig) {
	if config.TopN <= 0 {
		config.TopN = 5
	}
	if config.TopN > 20 {
		config.TopN = 20
	}
	config.PDFFallback = strings.ToLower(strings.TrimSpace(config.PDFFallback))
	if config.PDFFallback == "" {
		config.PDFFallback = "open_access"
	}
	if config.PDFFallback != "open_access" && config.PDFFallback != "none" {
		config.PDFFallback = "open_access"
	}
	if config.MaxPDFBytes <= 0 {
		config.MaxPDFBytes = 10 * 1024 * 1024
	}
	if config.PDFTextMaxChars <= 0 {
		config.PDFTextMaxChars = 12000
	}
	if config.MaxInputChars <= 0 {
		config.MaxInputChars = 16000
	}
	if config.TimeoutSeconds <= 0 {
		config.TimeoutSeconds = 60
	}
	if strings.TrimSpace(config.LLM.BaseURL) == "" {
		config.LLM.BaseURL = "https://api.deepseek.com/v1"
	}
	if strings.TrimSpace(config.LLM.Model) == "" {
		config.LLM.Model = "deepseek-v4-flash"
	}
	if config.LLM.MaxTokens <= 0 {
		config.LLM.MaxTokens = 1200
	}
	if config.LLM.MaxConcurrency <= 0 {
		config.LLM.MaxConcurrency = 5
	}
	if strings.TrimSpace(config.LLM.APIKey) == "" {
		config.Enabled = false
	}
}

func (config RequestConfig) sourcePatch() *SourceConfigPatch {
	if config.Timeout == nil && config.RequestTimeout == nil && config.MaxRetries == nil && config.RateLimit == nil {
		return nil
	}

	return &SourceConfigPatch{
		Timeout:        config.Timeout,
		RequestTimeout: config.RequestTimeout,
		MaxRetries:     config.MaxRetries,
		RateLimit:      config.RateLimit,
	}
}

func (patch SourceConfigPatch) apply(config SourceConfig) SourceConfig {
	patched, _ := patch.applyWithError(config)
	return patched
}

func (patch SourceConfigPatch) applyWithError(config SourceConfig) (SourceConfig, error) {
	if patch.Enabled != nil {
		config.Enabled = *patch.Enabled
	}
	if patch.APIKey != nil {
		config.APIKey = *patch.APIKey
	}
	if patch.Email != nil {
		config.APIKey = *patch.Email
	}
	if patch.BaseURL != nil {
		config.BaseURL = *patch.BaseURL
	}
	if len(patch.Feeds) > 0 {
		feeds, err := rawConfigValueString(patch.Feeds)
		if err != nil {
			return config, err
		}
		config.BaseURL = feeds
	}
	if patch.Timeout != nil {
		config.RequestTimeout = *patch.Timeout
	}
	if patch.RequestTimeout != nil {
		config.RequestTimeout = *patch.RequestTimeout
	}
	if patch.MaxRetries != nil {
		config.MaxRetries = *patch.MaxRetries
	}
	if patch.RateLimit != nil {
		config.RateLimit = *patch.RateLimit
	}

	return config, nil
}

func rawConfigValueString(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", nil
	}

	var text string
	if err := json.Unmarshal(trimmed, &text); err == nil {
		return text, nil
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, trimmed); err != nil {
		return "", err
	}
	return compact.String(), nil
}

func findProjectConfigFile(fileName string) (string, bool, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", false, err
	}

	for {
		path := filepath.Join(currentDir, fileName)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, true, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", false, err
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			return "", false, nil
		}
		currentDir = parent
	}
}

func loadDotEnvFromFile(fileName string) error {
	path, ok, err := findProjectConfigFile(fileName)
	if err != nil || !ok {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	values, err := parseDotEnv(content)
	if err != nil {
		return fmt.Errorf("解析.env文件 %s 失败: %w", path, err)
	}
	for key, value := range values {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("设置.env变量 %s 失败: %w", key, err)
		}
	}

	return nil
}

func parseDotEnv(content []byte) (map[string]string, error) {
	values := make(map[string]string)
	lines := strings.Split(string(content), "\n")
	for index, rawLine := range lines {
		key, value, ok, err := parseDotEnvLine(rawLine)
		if err != nil {
			return nil, fmt.Errorf("第%d行: %w", index+1, err)
		}
		if ok {
			values[key] = value
		}
	}
	return values, nil
}

func parseDotEnvLine(rawLine string) (string, string, bool, error) {
	line := strings.TrimSpace(rawLine)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false, nil
	}
	line = trimDotEnvExport(line)

	equalsIndex := strings.Index(line, "=")
	if equalsIndex <= 0 {
		return "", "", false, fmt.Errorf("缺少KEY=VALUE格式")
	}

	key := strings.TrimSpace(line[:equalsIndex])
	if !isDotEnvKey(key) {
		return "", "", false, fmt.Errorf("环境变量名无效: %s", key)
	}

	value, err := parseDotEnvValue(strings.TrimSpace(line[equalsIndex+1:]))
	if err != nil {
		return "", "", false, err
	}
	return key, value, true, nil
}

func trimDotEnvExport(line string) string {
	if !strings.HasPrefix(line, "export") || len(line) == len("export") {
		return line
	}
	if line[len("export")] != ' ' && line[len("export")] != '\t' {
		return line
	}
	return strings.TrimSpace(line[len("export"):])
}

func parseDotEnvValue(rawValue string) (string, error) {
	if rawValue == "" {
		return "", nil
	}
	switch rawValue[0] {
	case '"':
		return parseDoubleQuotedDotEnvValue(rawValue)
	case '\'':
		return parseSingleQuotedDotEnvValue(rawValue)
	default:
		return stripDotEnvInlineComment(rawValue), nil
	}
}

func parseDoubleQuotedDotEnvValue(rawValue string) (string, error) {
	closingIndex := closingDotEnvQuoteIndex(rawValue, '"', true)
	if closingIndex < 0 {
		return "", fmt.Errorf("双引号未闭合")
	}
	if err := validateDotEnvValueTrailing(rawValue[closingIndex+1:]); err != nil {
		return "", err
	}
	value, err := strconv.Unquote(rawValue[:closingIndex+1])
	if err != nil {
		return "", err
	}
	return value, nil
}

func parseSingleQuotedDotEnvValue(rawValue string) (string, error) {
	closingIndex := closingDotEnvQuoteIndex(rawValue, '\'', false)
	if closingIndex < 0 {
		return "", fmt.Errorf("单引号未闭合")
	}
	if err := validateDotEnvValueTrailing(rawValue[closingIndex+1:]); err != nil {
		return "", err
	}
	return rawValue[1:closingIndex], nil
}

func closingDotEnvQuoteIndex(rawValue string, quote byte, allowEscapes bool) int {
	escaped := false
	for index := 1; index < len(rawValue); index++ {
		current := rawValue[index]
		if allowEscapes && escaped {
			escaped = false
			continue
		}
		if allowEscapes && current == '\\' {
			escaped = true
			continue
		}
		if current == quote {
			return index
		}
	}
	return -1
}

func validateDotEnvValueTrailing(trailing string) error {
	trimmed := strings.TrimSpace(trailing)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}
	return fmt.Errorf("引号后存在非注释内容")
}

func stripDotEnvInlineComment(rawValue string) string {
	for index := 0; index < len(rawValue); index++ {
		if rawValue[index] == '#' && (index == 0 || rawValue[index-1] == ' ' || rawValue[index-1] == '\t') {
			return strings.TrimSpace(rawValue[:index])
		}
	}
	return strings.TrimSpace(rawValue)
}

func isDotEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for index, character := range key {
		if index == 0 {
			if !isDotEnvKeyStart(character) {
				return false
			}
			continue
		}
		if !isDotEnvKeyPart(character) {
			return false
		}
	}
	return true
}

func isDotEnvKeyStart(character rune) bool {
	return character == '_' || character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z'
}

func isDotEnvKeyPart(character rune) bool {
	return isDotEnvKeyStart(character) || character >= '0' && character <= '9'
}

func jsoncToJSON(content []byte) ([]byte, error) {
	withoutComments, err := stripJSONCComments(content)
	if err != nil {
		return nil, err
	}

	return removeTrailingJSONCommas(withoutComments), nil
}

func stripJSONCComments(content []byte) ([]byte, error) {
	scanner := newJSONCCommentScanner()
	for index := 0; index < len(content); index++ {
		index = scanner.consume(content, index)
	}

	return scanner.finish()
}

type jsoncScannerState int

const (
	jsoncDefaultState jsoncScannerState = iota
	jsoncStringState
	jsoncLineCommentState
	jsoncBlockCommentState
)

type jsoncCommentScanner struct {
	output  bytes.Buffer
	state   jsoncScannerState
	escaped bool
}

func newJSONCCommentScanner() *jsoncCommentScanner {
	return &jsoncCommentScanner{state: jsoncDefaultState}
}

func (scanner *jsoncCommentScanner) consume(content []byte, index int) int {
	switch scanner.state {
	case jsoncLineCommentState:
		return scanner.consumeLineComment(content, index)
	case jsoncBlockCommentState:
		return scanner.consumeBlockComment(content, index)
	case jsoncStringState:
		return scanner.consumeString(content, index)
	default:
		return scanner.consumeDefault(content, index)
	}
}

func (scanner *jsoncCommentScanner) consumeDefault(content []byte, index int) int {
	current := content[index]
	next := byteAt(content, index+1)

	switch {
	case current == '"':
		scanner.state = jsoncStringState
		scanner.output.WriteByte(current)
	case current == '/' && next == '/':
		scanner.state = jsoncLineCommentState
		return index + 1
	case current == '/' && next == '*':
		scanner.state = jsoncBlockCommentState
		return index + 1
	default:
		scanner.output.WriteByte(current)
	}

	return index
}

func (scanner *jsoncCommentScanner) consumeString(content []byte, index int) int {
	current := content[index]
	scanner.output.WriteByte(current)
	if scanner.escaped {
		scanner.escaped = false
		return index
	}

	if current == '\\' {
		scanner.escaped = true
	} else if current == '"' {
		scanner.state = jsoncDefaultState
	}
	return index
}

func (scanner *jsoncCommentScanner) consumeLineComment(content []byte, index int) int {
	current := content[index]
	if current == '\n' || current == '\r' {
		scanner.state = jsoncDefaultState
		scanner.output.WriteByte(current)
	}
	return index
}

func (scanner *jsoncCommentScanner) consumeBlockComment(content []byte, index int) int {
	current := content[index]
	next := byteAt(content, index+1)
	if current == '*' && next == '/' {
		scanner.state = jsoncDefaultState
		return index + 1
	}
	if current == '\n' || current == '\r' {
		scanner.output.WriteByte(current)
	}
	return index
}

func (scanner *jsoncCommentScanner) finish() ([]byte, error) {
	if scanner.state == jsoncBlockCommentState {
		return nil, fmt.Errorf("JSONC块注释未闭合")
	}
	if scanner.state == jsoncStringState {
		return nil, fmt.Errorf("JSONC字符串未闭合")
	}

	return scanner.output.Bytes(), nil
}

func removeTrailingJSONCommas(content []byte) []byte {
	var output bytes.Buffer
	inString := false
	escaped := false

	for index := 0; index < len(content); index++ {
		current := content[index]
		if inString {
			output.WriteByte(current)
			if escaped {
				escaped = false
				continue
			}
			if current == '\\' {
				escaped = true
			} else if current == '"' {
				inString = false
			}
			continue
		}

		if current == '"' {
			inString = true
			output.WriteByte(current)
			continue
		}
		if current == ',' && isClosingJSONToken(nextNonWhitespace(content, index+1)) {
			continue
		}
		output.WriteByte(current)
	}

	return output.Bytes()
}

func isClosingJSONToken(value byte) bool {
	return value == '}' || value == ']'
}

func nextNonWhitespace(content []byte, start int) byte {
	for index := start; index < len(content); index++ {
		switch content[index] {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return content[index]
		}
	}
	return 0
}

func byteAt(content []byte, index int) byte {
	if index >= len(content) {
		return 0
	}
	return content[index]
}

func boolEnv(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func intEnv(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func int64Env(key string, defaultValue int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func floatEnv(key string, defaultValue float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return parsed
}
