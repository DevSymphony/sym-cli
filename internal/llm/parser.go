package llm

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"regexp"
	"strings"
)

// internalFormat is the internal response format type used by the parser.
type internalFormat int

const (
	ResponseFormatText internalFormat = iota
	ResponseFormatJSON
	ResponseFormatXML
)

func (f internalFormat) String() string {
	switch f {
	case ResponseFormatJSON:
		return "json"
	case ResponseFormatXML:
		return "xml"
	default:
		return "text"
	}
}

// ParseOptions configures response parsing behavior.
type ParseOptions struct {
	Format     internalFormat
	StrictMode bool // If true, return error when format not found
}

var (
	ErrNoJSONFound = errors.New("no valid JSON found in response")
	ErrNoXMLFound  = errors.New("no valid XML found in response")
)

// ParseResponse extracts structured content from LLM responses.
// Handles cases where LLM adds preamble text like "I need to analyze..."
func ParseResponse(response string, opts ParseOptions) (string, error) {
	switch opts.Format {
	case ResponseFormatJSON:
		return extractJSON(response, opts.StrictMode)
	case ResponseFormatXML:
		return extractXML(response, opts.StrictMode)
	default:
		return response, nil
	}
}

// extractJSON finds and extracts JSON content from response.
func extractJSON(response string, strict bool) (string, error) {
	// Strategy 1: Look for code block with json marker
	if jsonBlock := extractCodeBlock(response, "json"); jsonBlock != "" {
		if isValidJSON(jsonBlock) {
			return jsonBlock, nil
		}
	}

	// Strategy 2: Find outermost { } or [ ]
	if jsonStr := findJSONBoundaries(response); jsonStr != "" {
		if isValidJSON(jsonStr) {
			return jsonStr, nil
		}
	}

	// Strategy 3: Try entire response as JSON
	trimmed := strings.TrimSpace(response)
	if isValidJSON(trimmed) {
		return trimmed, nil
	}

	if strict {
		return "", ErrNoJSONFound
	}
	return response, nil
}

// extractXML finds and extracts XML content from response.
func extractXML(response string, strict bool) (string, error) {
	// Strategy 1: Look for code block with xml marker
	if xmlBlock := extractCodeBlock(response, "xml"); xmlBlock != "" {
		if isValidXML(xmlBlock) {
			return xmlBlock, nil
		}
	}

	// Strategy 2: Find <?xml or first < tag to last > tag
	if xmlStr := findXMLBoundaries(response); xmlStr != "" {
		if isValidXML(xmlStr) {
			return xmlStr, nil
		}
	}

	// Strategy 3: Try entire response as XML
	trimmed := strings.TrimSpace(response)
	if isValidXML(trimmed) {
		return trimmed, nil
	}

	if strict {
		return "", ErrNoXMLFound
	}
	return response, nil
}

// extractCodeBlock matches ```lang ... ``` code blocks.
func extractCodeBlock(response, lang string) string {
	patternStr := "(?s)```" + regexp.QuoteMeta(lang) + "\\s*\\n?(.*?)```"
	pattern := regexp.MustCompile(patternStr)
	matches := pattern.FindStringSubmatch(response)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// findJSONBoundaries finds first { or [ and matches to corresponding } or ].
func findJSONBoundaries(s string) string {
	start := -1
	var startChar byte

	for i := 0; i < len(s); i++ {
		if s[i] == '{' || s[i] == '[' {
			start = i
			startChar = s[i]
			break
		}
	}

	if start == -1 {
		return ""
	}

	var endChar byte
	if startChar == '{' {
		endChar = '}'
	} else {
		endChar = ']'
	}

	// Find matching end (handle nesting)
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		c := s[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == startChar {
			depth++
		} else if c == endChar {
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	return ""
}

// findXMLBoundaries finds <?xml or first < and last >.
func findXMLBoundaries(s string) string {
	// Look for <?xml declaration first
	start := strings.Index(s, "<?xml")
	if start == -1 {
		// Find first opening tag
		start = strings.Index(s, "<")
	}
	if start == -1 {
		return ""
	}

	end := strings.LastIndex(s, ">")
	if end <= start {
		return ""
	}

	return s[start : end+1]
}

func isValidJSON(s string) bool {
	var js interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}

func isValidXML(s string) bool {
	// Check if string starts with < (basic XML requirement)
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 || trimmed[0] != '<' {
		return false
	}

	// Simple XML validation using decoder
	decoder := xml.NewDecoder(strings.NewReader(s))
	tokenCount := 0
	for {
		_, err := decoder.Token()
		if err != nil {
			// Valid XML must have at least one token
			return err.Error() == "EOF" && tokenCount > 0
		}
		tokenCount++
	}
}
