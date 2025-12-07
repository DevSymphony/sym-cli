package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInternalFormatString(t *testing.T) {
	tests := []struct {
		format   internalFormat
		expected string
	}{
		{ResponseFormatText, "text"},
		{ResponseFormatJSON, "json"},
		{ResponseFormatXML, "xml"},
		{internalFormat(99), "text"}, // unknown defaults to text
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.format.String())
		})
	}
}

func TestResponseFormatString(t *testing.T) {
	assert.Equal(t, "text", Text.String())
	assert.Equal(t, "json", JSON.String())
	assert.Equal(t, "xml", XML.String())
}

func TestParseResponseText(t *testing.T) {
	t.Run("returns response as-is for text format", func(t *testing.T) {
		response := "This is plain text response."
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatText})

		require.NoError(t, err)
		assert.Equal(t, response, result)
	})
}

func TestParseResponseJSON(t *testing.T) {
	t.Run("extracts JSON from code block", func(t *testing.T) {
		response := "Here is the result:\n```json\n{\"key\": \"value\"}\n```"
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatJSON})

		require.NoError(t, err)
		assert.Equal(t, `{"key": "value"}`, result)
	})

	t.Run("extracts JSON object from mixed content", func(t *testing.T) {
		response := `I analyzed the code and found: {"result": true, "count": 5} Hope this helps!`
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatJSON})

		require.NoError(t, err)
		assert.Equal(t, `{"result": true, "count": 5}`, result)
	})

	t.Run("extracts JSON array from mixed content", func(t *testing.T) {
		response := `The linters are: ["eslint", "prettier"]`
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatJSON})

		require.NoError(t, err)
		assert.Equal(t, `["eslint", "prettier"]`, result)
	})

	t.Run("handles nested JSON objects", func(t *testing.T) {
		response := `{"outer": {"inner": {"deep": "value"}}}`
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatJSON})

		require.NoError(t, err)
		assert.Equal(t, response, result)
	})

	t.Run("handles JSON with escaped characters", func(t *testing.T) {
		response := `{"message": "Hello \"World\"", "path": "C:\\Users"}`
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatJSON})

		require.NoError(t, err)
		assert.Equal(t, response, result)
	})

	t.Run("returns original on no JSON in non-strict mode", func(t *testing.T) {
		response := "No JSON here, just text."
		result, err := ParseResponse(response, ParseOptions{
			Format:     ResponseFormatJSON,
			StrictMode: false,
		})

		require.NoError(t, err)
		assert.Equal(t, response, result)
	})

	t.Run("returns error on no JSON in strict mode", func(t *testing.T) {
		response := "No JSON here, just text."
		_, err := ParseResponse(response, ParseOptions{
			Format:     ResponseFormatJSON,
			StrictMode: true,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoJSONFound)
	})

	t.Run("handles empty JSON object", func(t *testing.T) {
		response := `{}`
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatJSON})

		require.NoError(t, err)
		assert.Equal(t, `{}`, result)
	})

	t.Run("handles empty JSON array", func(t *testing.T) {
		response := `[]`
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatJSON})

		require.NoError(t, err)
		assert.Equal(t, `[]`, result)
	})
}

func TestParseResponseXML(t *testing.T) {
	t.Run("extracts XML from code block", func(t *testing.T) {
		response := "Here is the XML:\n```xml\n<root><item>value</item></root>\n```"
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatXML})

		require.NoError(t, err)
		assert.Equal(t, `<root><item>value</item></root>`, result)
	})

	t.Run("extracts XML with declaration", func(t *testing.T) {
		response := `Some preamble. <?xml version="1.0"?><root><data>test</data></root> Some epilogue.`
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatXML})

		require.NoError(t, err)
		assert.Contains(t, result, `<?xml version="1.0"?>`)
		assert.Contains(t, result, `<root>`)
	})

	t.Run("extracts XML without declaration", func(t *testing.T) {
		response := `Analysis complete: <config><setting name="test">value</setting></config>`
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatXML})

		require.NoError(t, err)
		assert.Equal(t, `<config><setting name="test">value</setting></config>`, result)
	})

	t.Run("returns original on no XML in non-strict mode", func(t *testing.T) {
		response := "No XML here."
		result, err := ParseResponse(response, ParseOptions{
			Format:     ResponseFormatXML,
			StrictMode: false,
		})

		require.NoError(t, err)
		assert.Equal(t, response, result)
	})

	t.Run("returns error on no XML in strict mode", func(t *testing.T) {
		response := "No XML here."
		_, err := ParseResponse(response, ParseOptions{
			Format:     ResponseFormatXML,
			StrictMode: true,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNoXMLFound)
	})

	t.Run("handles self-closing XML tags", func(t *testing.T) {
		response := `<root><empty/></root>`
		result, err := ParseResponse(response, ParseOptions{Format: ResponseFormatXML})

		require.NoError(t, err)
		assert.Equal(t, response, result)
	})
}

func TestExtractCodeBlock(t *testing.T) {
	t.Run("extracts json code block", func(t *testing.T) {
		response := "```json\n{\"key\": \"value\"}\n```"
		result := extractCodeBlock(response, "json")
		assert.Equal(t, `{"key": "value"}`, result)
	})

	t.Run("extracts xml code block", func(t *testing.T) {
		response := "```xml\n<root>value</root>\n```"
		result := extractCodeBlock(response, "xml")
		assert.Equal(t, `<root>value</root>`, result)
	})

	t.Run("returns empty for no match", func(t *testing.T) {
		response := "```python\nprint('hello')\n```"
		result := extractCodeBlock(response, "json")
		assert.Empty(t, result)
	})

	t.Run("handles code block without newlines", func(t *testing.T) {
		response := "```json{\"key\": \"value\"}```"
		result := extractCodeBlock(response, "json")
		assert.Equal(t, `{"key": "value"}`, result)
	})
}

func TestFindJSONBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple object",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "simple array",
			input:    `["a", "b"]`,
			expected: `["a", "b"]`,
		},
		{
			name:     "nested objects",
			input:    `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:     "with preamble",
			input:    `Here is JSON: {"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "with epilogue",
			input:    `{"key": "value"} Hope this helps!`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "with braces in string",
			input:    `{"msg": "use { and } carefully"}`,
			expected: `{"msg": "use { and } carefully"}`,
		},
		{
			name:     "no JSON",
			input:    `Just plain text`,
			expected: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findJSONBoundaries(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindXMLBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple element",
			input:    `<root>value</root>`,
			expected: `<root>value</root>`,
		},
		{
			name:     "with declaration",
			input:    `<?xml version="1.0"?><root/>`,
			expected: `<?xml version="1.0"?><root/>`,
		},
		{
			name:     "with preamble",
			input:    `Here is XML: <config>test</config>`,
			expected: `<config>test</config>`,
		},
		{
			name:     "no XML",
			input:    `No XML here`,
			expected: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findXMLBoundaries(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidJSON(t *testing.T) {
	assert.True(t, isValidJSON(`{"key": "value"}`))
	assert.True(t, isValidJSON(`["a", "b"]`))
	assert.True(t, isValidJSON(`null`))
	assert.True(t, isValidJSON(`123`))
	assert.True(t, isValidJSON(`"string"`))

	assert.False(t, isValidJSON(`{invalid}`))
	assert.False(t, isValidJSON(`not json`))
	assert.False(t, isValidJSON(``))
}

func TestIsValidXML(t *testing.T) {
	assert.True(t, isValidXML(`<root>value</root>`))
	assert.True(t, isValidXML(`<empty/>`))
	assert.True(t, isValidXML(`<?xml version="1.0"?><root/>`))

	assert.False(t, isValidXML(`not xml`))
	assert.False(t, isValidXML(`<unclosed>`))
	assert.False(t, isValidXML(``))
}
