package checkstyle

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/DevSymphony/sym-cli/internal/linter"
	"github.com/DevSymphony/sym-cli/internal/llm"
	"github.com/DevSymphony/sym-cli/pkg/schema"
)

// Compile-time interface check
var _ linter.Converter = (*Converter)(nil)

// Converter converts rules to Checkstyle XML configuration using LLM
type Converter struct{}

// NewConverter creates a new Checkstyle converter
func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) Name() string {
	return "checkstyle"
}

func (c *Converter) SupportedLanguages() []string {
	return []string{"java"}
}

// GetLLMDescription returns a description of Checkstyle's capabilities for LLM routing
func (c *Converter) GetLLMDescription() string {
	return `Java style checks (naming, whitespace, imports, line length, complexity)
  - CAN: Class/method/variable naming, line/method length, indentation, import checks, cyclomatic complexity, JavaDoc
  - CANNOT: Runtime behavior, business logic, security vulnerabilities, advanced design patterns`
}

// GetRoutingHints returns routing rules for LLM to decide when to use Checkstyle
func (c *Converter) GetRoutingHints() []string {
	return []string{
		"For Java naming rules (class names, variable names, method names) → ALWAYS use checkstyle",
		"For Java formatting rules (line length, indentation, whitespace) → use checkstyle",
		"For Java import rules (star imports, unused imports) → use checkstyle",
	}
}

type checkstyleModule struct {
	XMLName    xml.Name             `xml:"module"`
	Name       string               `xml:"name,attr"`
	Properties []checkstyleProperty `xml:"property,omitempty"`
	Modules    []checkstyleModule   `xml:"module,omitempty"`
}

type checkstyleProperty struct {
	XMLName xml.Name `xml:"property"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
}

type checkstyleConfig struct {
	XMLName xml.Name           `xml:"module"`
	Name    string             `xml:"name,attr"`
	Modules []checkstyleModule `xml:"module"`
}

// ConvertSingleRule converts ONE user rule to Checkstyle module.
// Returns (result, nil) on success,
//
//	(nil, nil) if rule cannot be converted by Checkstyle (skip),
//	(nil, error) on actual conversion error.
//
// Note: Concurrency is handled by the main converter.
func (c *Converter) ConvertSingleRule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*linter.SingleRuleResult, error) {
	if provider == nil {
		return nil, fmt.Errorf("LLM provider is required")
	}

	module, err := c.convertToCheckstyleModule(ctx, rule, provider)
	if err != nil {
		return nil, err
	}

	if module == nil {
		return nil, nil
	}

	return &linter.SingleRuleResult{
		RuleID: rule.ID,
		Data:   module,
	}, nil
}

// BuildConfig assembles Checkstyle XML configuration from successful rule conversions.
func (c *Converter) BuildConfig(results []*linter.SingleRuleResult) (*linter.LinterConfig, error) {
	if len(results) == 0 {
		return nil, nil
	}

	var modules []checkstyleModule
	for _, r := range results {
		module, ok := r.Data.(*checkstyleModule)
		if !ok {
			continue
		}
		modules = append(modules, *module)
	}

	if len(modules) == 0 {
		return nil, nil
	}

	// Separate modules into Checker-level and TreeWalker-level
	checkerLevelModules := map[string]bool{
		"LineLength":                         true,
		"FileLength":                         true,
		"FileTabCharacter":                   true,
		"NewlineAtEndOfFile":                 true,
		"UniqueProperties":                   true,
		"OrderedProperties":                  true,
		"Translation":                        true,
		"SuppressWarningsFilter":             true,
		"BeforeExecutionExclusionFileFilter": true,
		"SuppressionFilter":                  true,
		"SuppressionCommentFilter":           true,
	}

	var checkerModules []checkstyleModule
	var treeWalkerModules []checkstyleModule

	for _, module := range modules {
		if checkerLevelModules[module.Name] {
			checkerModules = append(checkerModules, module)
		} else {
			treeWalkerModules = append(treeWalkerModules, module)
		}
	}

	// Build Checkstyle configuration
	treeWalker := checkstyleModule{
		Name:    "TreeWalker",
		Modules: treeWalkerModules,
	}

	allModules := append(checkerModules, treeWalker)
	config := checkstyleConfig{
		Name:    "Checker",
		Modules: allModules,
	}

	// Marshal to XML
	content, err := xml.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add XML header
	xmlHeader := `<?xml version="1.0"?>
<!DOCTYPE module PUBLIC
    "-//Checkstyle//DTD Checkstyle Configuration 1.3//EN"
    "https://checkstyle.org/dtds/configuration_1_3.dtd">
`
	fullContent := []byte(xmlHeader + string(content))

	return &linter.LinterConfig{
		Filename: "checkstyle.xml",
		Content:  fullContent,
		Format:   "xml",
	}, nil
}

// convertToCheckstyleModule converts a single rule using LLM
func (c *Converter) convertToCheckstyleModule(ctx context.Context, rule schema.UserRule, provider llm.Provider) (*checkstyleModule, error) {
	systemPrompt := `You are a Checkstyle configuration expert. Convert natural language Java coding rules to Checkstyle modules.

Return ONLY a JSON object (no markdown fences):
{
  "module_name": "CheckstyleModuleName",
  "severity": "error|warning|info",
  "properties": {"key": "value", ...}
}

Common Checkstyle modules:
- Naming: TypeName, MethodName, MemberName, ParameterName, LocalVariableName, StaticVariableName, ConstantName
- Length: LineLength, MethodLength, ParameterNumber, FileLength
- Style: Indentation, WhitespaceAround, NeedBraces, LeftCurly, RightCurly
- Imports: AvoidStarImport, IllegalImport, UnusedImports
- Complexity: CyclomaticComplexity, NPathComplexity
- JavaDoc: JavadocMethod, JavadocType, MissingJavadocMethod

IMPORTANT - Use MemberName for class fields (instance variables), NOT LocalVariableName:
- MemberName: private/protected/public instance variables (class fields)
- LocalVariableName: variables declared inside methods (local scope only)
- StaticVariableName: static non-final variables

If cannot convert, return:
{
  "module_name": "",
  "severity": "error",
  "properties": {}
}

Examples:

Input: "Methods must not exceed 50 lines"
Output:
{
  "module_name": "MethodLength",
  "severity": "error",
  "properties": {"max": "50"}
}

Input: "Use camelCase for local variables"
Output:
{
  "module_name": "LocalVariableName",
  "severity": "error",
  "properties": {"format": "^[a-z][a-zA-Z0-9]*$"}
}

Input: "Private member variables must start with m_"
Output:
{
  "module_name": "MemberName",
  "severity": "error",
  "properties": {"format": "^m_[a-z][a-zA-Z0-9]*$"}
}

Input: "Class names must be PascalCase"
Output:
{
  "module_name": "TypeName",
  "severity": "error",
  "properties": {"format": "^[A-Z][a-zA-Z0-9]*$"}
}

Input: "Method names must be camelCase"
Output:
{
  "module_name": "MethodName",
  "severity": "error",
  "properties": {"format": "^[a-z][a-zA-Z0-9]*$"}
}`

	userPrompt := fmt.Sprintf("Convert this Java rule to Checkstyle module:\n\n%s", rule.Say)

	// Call LLM
	prompt := systemPrompt + "\n\n" + userPrompt
	response, err := provider.Execute(ctx, prompt, llm.JSON)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	response = linter.CleanJSONResponse(response)

	if response == "" {
		return nil, fmt.Errorf("LLM returned empty response")
	}

	var result struct {
		ModuleName string            `json:"module_name"`
		Severity   string            `json:"severity"`
		Properties map[string]string `json:"properties"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w (response: %.100s)", err, response)
	}

	if result.ModuleName == "" {
		return nil, nil
	}

	// Filter properties to only include valid ones for this module
	filteredProps := filterValidProperties(result.ModuleName, result.Properties)

	// Build module
	module := &checkstyleModule{
		Name:       result.ModuleName,
		Properties: []checkstyleProperty{},
	}

	// Add severity
	module.Properties = append(module.Properties, checkstyleProperty{
		Name:  "severity",
		Value: mapCheckstyleSeverity(result.Severity),
	})

	// Add filtered properties
	for key, value := range filteredProps {
		if key == "severity" {
			continue // Already added above
		}
		module.Properties = append(module.Properties, checkstyleProperty{
			Name:  key,
			Value: value,
		})
	}

	return module, nil
}

// mapCheckstyleSeverity maps severity to Checkstyle severity
func mapCheckstyleSeverity(severity string) string {
	switch strings.ToLower(severity) {
	case "error":
		return "error"
	case "warning", "warn":
		return "warning"
	case "info":
		return "info"
	default:
		return "error"
	}
}

// validCheckstyleProperties defines valid properties for each Checkstyle module
var validCheckstyleProperties = map[string]map[string]bool{
	"TypeName": {
		"severity": true,
		"format":   true,
		"tokens":   true,
	},
	"MethodName": {
		"severity":         true,
		"format":           true,
		"allowClassName":   true,
		"applyToPublic":    true,
		"applyToProtected": true,
		"applyToPackage":   true,
		"applyToPrivate":   true,
		"tokens":           true,
	},
	"ParameterName": {
		"severity":         true,
		"format":           true,
		"ignoreOverridden": true,
		"accessModifiers":  true,
	},
	"LocalVariableName": {
		"severity":                 true,
		"format":                   true,
		"allowOneCharVarInForLoop": true,
	},
	"ConstantName": {
		"severity":         true,
		"format":           true,
		"applyToPublic":    true,
		"applyToProtected": true,
		"applyToPackage":   true,
		"applyToPrivate":   true,
	},
	"LineLength": {
		"severity":       true,
		"max":            true,
		"ignorePattern":  true,
		"fileExtensions": true,
	},
	"MethodLength": {
		"severity":   true,
		"max":        true,
		"countEmpty": true,
		"tokens":     true,
	},
	"ParameterNumber": {
		"severity":                true,
		"max":                     true,
		"ignoreOverriddenMethods": true,
		"tokens":                  true,
	},
	"FileLength": {
		"severity":       true,
		"max":            true,
		"fileExtensions": true,
	},
	"Indentation": {
		"severity":                true,
		"basicOffset":             true,
		"braceAdjustment":         true,
		"caseIndent":              true,
		"throwsIndent":            true,
		"arrayInitIndent":         true,
		"lineWrappingIndentation": true,
		"forceStrictCondition":    true,
	},
	"WhitespaceAround": {
		"severity":               true,
		"allowEmptyConstructors": true,
		"allowEmptyMethods":      true,
		"allowEmptyTypes":        true,
		"allowEmptyLoops":        true,
		"allowEmptyLambdas":      true,
		"allowEmptyCatches":      true,
		"ignoreEnhancedForColon": true,
		"tokens":                 true,
	},
	"NeedBraces": {
		"severity":                 true,
		"allowSingleLineStatement": true,
		"allowEmptyLoopBody":       true,
		"tokens":                   true,
	},
	"LeftCurly": {
		"severity":    true,
		"option":      true,
		"ignoreEnums": true,
		"tokens":      true,
	},
	"RightCurly": {
		"severity": true,
		"option":   true,
		"tokens":   true,
	},
	"AvoidStarImport": {
		"severity":                 true,
		"excludes":                 true,
		"allowClassImports":        true,
		"allowStaticMemberImports": true,
	},
	"IllegalImport": {
		"severity":       true,
		"illegalPkgs":    true,
		"illegalClasses": true,
		"regexp":         true,
	},
	"UnusedImports": {
		"severity":       true,
		"processJavadoc": true,
	},
	"CyclomaticComplexity": {
		"severity":                         true,
		"max":                              true,
		"switchBlockAsSingleDecisionPoint": true,
		"tokens":                           true,
	},
	"NPathComplexity": {
		"severity": true,
		"max":      true,
	},
	"JavadocMethod": {
		"severity":              true,
		"accessModifiers":       true,
		"allowMissingParamTags": true,
		"allowMissingReturnTag": true,
		"allowedAnnotations":    true,
		"validateThrows":        true,
		"tokens":                true,
	},
	"JavadocType": {
		"severity":              true,
		"scope":                 true,
		"excludeScope":          true,
		"authorFormat":          true,
		"versionFormat":         true,
		"allowMissingParamTags": true,
		"allowUnknownTags":      true,
		"allowedAnnotations":    true,
		"tokens":                true,
	},
	"MissingJavadocMethod": {
		"severity":                    true,
		"minLineCount":                true,
		"allowedAnnotations":          true,
		"scope":                       true,
		"excludeScope":                true,
		"allowMissingPropertyJavadoc": true,
		"ignoreMethodNamesRegex":      true,
		"tokens":                      true,
	},
	"EmptyBlock": {
		"severity": true,
		"option":   true,
		"tokens":   true,
	},
	"MagicNumber": {
		"severity":                        true,
		"ignoreNumbers":                   true,
		"ignoreHashCodeMethod":            true,
		"ignoreAnnotation":                true,
		"ignoreFieldDeclaration":          true,
		"ignoreAnnotationElementDefaults": true,
		"constantWaiverParentToken":       true,
		"tokens":                          true,
	},
}

// filterValidProperties filters out invalid properties for a given module
func filterValidProperties(moduleName string, properties map[string]string) map[string]string {
	validProps, hasDefinedProps := validCheckstyleProperties[moduleName]
	if !hasDefinedProps {
		result := make(map[string]string)
		if sev, ok := properties["severity"]; ok {
			result["severity"] = sev
		}
		return result
	}

	result := make(map[string]string)
	for key, value := range properties {
		if validProps[key] {
			result[key] = value
		}
	}
	return result
}
