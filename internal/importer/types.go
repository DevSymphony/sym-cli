package importer

import "github.com/DevSymphony/sym-cli/pkg/schema"

// ImportMode defines how imported conventions are merged with existing ones
type ImportMode string

const (
	// ImportModeAppend keeps existing categories/rules and adds new ones
	ImportModeAppend ImportMode = "append"
	// ImportModeClear removes existing categories/rules, then imports new ones
	ImportModeClear ImportMode = "clear"
)

// ImportInput represents the input for import operation
type ImportInput struct {
	Path string     // Single file path to import
	Mode ImportMode // Import mode (clear or append)
}

// ImportResult represents the result of import operation
type ImportResult struct {
	CategoriesAdded   []schema.CategoryDef // New categories added
	RulesAdded        []schema.UserRule    // New rules added
	CategoriesRemoved int                  // Categories removed (clear mode only)
	RulesRemoved      int                  // Rules removed (clear mode only)
	FileProcessed     string               // Processed file path
	Warnings          []string             // Non-fatal warnings
}

// DocumentContent represents parsed document content
type DocumentContent struct {
	Path    string // File path
	Content string // Extracted text content
	Format  string // Original format (txt, md, go, etc.)
	Size    int64  // Original file size
}

// ExtractedConventions represents LLM-extracted conventions from a document
type ExtractedConventions struct {
	Categories []schema.CategoryDef
	Rules      []schema.UserRule
	Source     string // Source document path
}

// LLMExtractionResponse represents the expected JSON response from LLM
type LLMExtractionResponse struct {
	Categories []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"categories"`
	Rules []struct {
		ID        string   `json:"id"`
		Say       string   `json:"say"`
		Category  string   `json:"category"`
		Languages []string `json:"languages,omitempty"`
		Severity  string   `json:"severity,omitempty"`
		Message   string   `json:"message,omitempty"`
		Example   string   `json:"example,omitempty"`
	} `json:"rules"`
}
