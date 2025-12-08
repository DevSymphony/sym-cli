package policy

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/DevSymphony/sym-cli/pkg/schema"
)

//go:embed templates/*.json
var templateFiles embed.FS

// Template represents a policy template metadata
type Template struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Framework   string `json:"framework,omitempty"`
}

// GetTemplates returns a list of available templates
func GetTemplates() ([]Template, error) {
	templates := []Template{}

	entries, err := fs.ReadDir(templateFiles, "templates")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		template := Template{
			Name: name,
		}

		// Set description based on template name
		switch name {
		case "demo-template":
			template.Description = "샘플 자바 정책 템플릿 (다양한 규칙 포함)"
			template.Language = "Java"
		case "react-template":
			template.Description = "React 프로젝트용 정책 (컴포넌트, Hooks, JSX 규칙)"
			template.Language = "JavaScript/TypeScript"
			template.Framework = "React"
		case "vue-template":
			template.Description = "Vue.js 프로젝트용 정책 (컴포넌트, Composition API)"
			template.Language = "JavaScript/TypeScript"
			template.Framework = "Vue.js"
		case "node-template":
			template.Description = "Node.js 백엔드용 정책 (Express, API, 에러 처리)"
			template.Language = "JavaScript/TypeScript"
			template.Framework = "Node.js"
		case "python-template":
			template.Description = "Python 프로젝트용 정책 (PEP 8, Django/Flask)"
			template.Language = "Python"
		case "typescript-template":
			template.Description = "TypeScript 라이브러리용 정책 (타입 안전성, 모듈)"
			template.Language = "TypeScript"
		case "go-template":
			template.Description = "Go 마이크로서비스용 정책 (네이밍, 에러 처리)"
			template.Language = "Go"
		default:
			template.Description = "사용자 정의 정책 템플릿"
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// GetTemplate returns the content of a specific template
func GetTemplate(name string) (*schema.UserPolicy, error) {
	fileName := name
	if !strings.HasSuffix(fileName, ".json") {
		fileName = name + ".json"
	}

	// Use path.Join instead of filepath.Join for embed.FS compatibility
	// embed.FS always uses forward slashes, regardless of OS
	filePath := path.Join("templates", fileName)
	data, err := templateFiles.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("template '%s' not found", name)
	}

	var policy schema.UserPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("invalid template file: %w", err)
	}

	return &policy, nil
}
