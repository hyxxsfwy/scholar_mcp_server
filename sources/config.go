package sources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultProjectConfigFile = "config.jsonc"

type ProjectConfig struct {
	Server  ServerConfig                 `json:"server"`
	Request RequestConfig                `json:"request"`
	Sources map[string]SourceConfigPatch `json:"sources"`
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
