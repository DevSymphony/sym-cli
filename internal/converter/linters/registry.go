package linters

// ESLintRuleRegistry contains all valid native ESLint rules with their options schema
// This is used to validate LLM-generated rules and prevent invalid configurations
var ESLintRuleRegistry = map[string]RuleDefinition{
	// Console/Debug
	"no-console": {
		Description: "Disallow the use of console",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"allow": {Type: "array", Items: "string"},
			},
		},
	},
	"no-debugger": {
		Description: "Disallow the use of debugger",
	},
	"no-alert": {
		Description: "Disallow the use of alert, confirm, and prompt",
	},

	// Variables
	"no-unused-vars": {
		Description: "Disallow unused variables",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"vars":               {Type: "string", Enum: []string{"all", "local"}},
				"varsIgnorePattern":  {Type: "string"},
				"args":               {Type: "string", Enum: []string{"all", "after-used", "none"}},
				"argsIgnorePattern":  {Type: "string"},
				"caughtErrors":       {Type: "string", Enum: []string{"all", "none"}},
				"ignoreRestSiblings": {Type: "boolean"},
			},
		},
	},
	"no-undef": {
		Description: "Disallow the use of undeclared variables",
	},
	"no-var": {
		Description: "Require let or const instead of var",
	},
	"prefer-const": {
		Description: "Require const declarations for variables that are never reassigned",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"destructuring":          {Type: "string", Enum: []string{"any", "all"}},
				"ignoreReadBeforeAssign": {Type: "boolean"},
			},
		},
	},

	// Naming
	"camelcase": {
		Description: "Enforce camelcase naming convention",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"properties":          {Type: "string", Enum: []string{"always", "never"}},
				"ignoreDestructuring": {Type: "boolean"},
				"ignoreImports":       {Type: "boolean"},
				"ignoreGlobals":       {Type: "boolean"},
				"allow":               {Type: "array", Items: "string"},
			},
		},
	},
	"new-cap": {
		Description: "Require constructor names to begin with a capital letter",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"newIsCap":           {Type: "boolean"},
				"capIsNew":           {Type: "boolean"},
				"newIsCapExceptions": {Type: "array", Items: "string"},
				"capIsNewExceptions": {Type: "array", Items: "string"},
				"properties":         {Type: "boolean"},
			},
		},
	},
	"id-length": {
		Description: "Enforce minimum and maximum identifier lengths",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"min":        {Type: "number"},
				"max":        {Type: "number"},
				"properties": {Type: "string", Enum: []string{"always", "never"}},
				"exceptions": {Type: "array", Items: "string"},
			},
		},
	},
	"id-match": {
		Description: "Require identifiers to match a specified regular expression",
		Options: OptionsSchema{
			Type: "string", // regex pattern
		},
	},

	// Code Quality
	"eqeqeq": {
		Description: "Require the use of === and !==",
		Options: OptionsSchema{
			Type: "string",
			Enum: []string{"always", "smart"},
		},
	},
	"no-eval": {
		Description: "Disallow the use of eval()",
	},
	"no-implied-eval": {
		Description: "Disallow the use of eval()-like methods",
	},
	"no-new-func": {
		Description: "Disallow new operators with the Function object",
	},

	// Complexity
	"complexity": {
		Description: "Enforce a maximum cyclomatic complexity",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"max": {Type: "number"},
			},
		},
	},
	"max-depth": {
		Description: "Enforce a maximum depth that blocks can be nested",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"max": {Type: "number"},
			},
		},
	},
	"max-nested-callbacks": {
		Description: "Enforce a maximum depth that callbacks can be nested",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"max": {Type: "number"},
			},
		},
	},

	// Length/Size
	"max-len": {
		Description: "Enforce a maximum line length",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"code":                   {Type: "number"},
				"tabWidth":               {Type: "number"},
				"comments":               {Type: "number"},
				"ignorePattern":          {Type: "string"},
				"ignoreComments":         {Type: "boolean"},
				"ignoreTrailingComments": {Type: "boolean"},
				"ignoreUrls":             {Type: "boolean"},
				"ignoreStrings":          {Type: "boolean"},
				"ignoreTemplateLiterals": {Type: "boolean"},
				"ignoreRegExpLiterals":   {Type: "boolean"},
			},
		},
	},
	"max-lines": {
		Description: "Enforce a maximum number of lines per file",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"max":            {Type: "number"},
				"skipBlankLines": {Type: "boolean"},
				"skipComments":   {Type: "boolean"},
			},
		},
	},
	"max-lines-per-function": {
		Description: "Enforce a maximum number of lines of code in a function",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"max":            {Type: "number"},
				"skipBlankLines": {Type: "boolean"},
				"skipComments":   {Type: "boolean"},
				"IIFEs":          {Type: "boolean"},
			},
		},
	},
	"max-params": {
		Description: "Enforce a maximum number of parameters in function definitions",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"max": {Type: "number"},
			},
		},
	},
	"max-statements": {
		Description: "Enforce a maximum number of statements allowed in function blocks",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"max":                     {Type: "number"},
				"ignoreTopLevelFunctions": {Type: "boolean"},
			},
		},
	},

	// Style
	"indent": {
		Description: "Enforce consistent indentation",
		Options: OptionsSchema{
			Type: "mixed", // number or "tab"
		},
	},
	"quotes": {
		Description: "Enforce the consistent use of either backticks, double, or single quotes",
		Options: OptionsSchema{
			Type: "string",
			Enum: []string{"single", "double", "backtick"},
		},
	},
	"semi": {
		Description: "Require or disallow semicolons instead of ASI",
		Options: OptionsSchema{
			Type: "string",
			Enum: []string{"always", "never"},
		},
	},
	"comma-dangle": {
		Description: "Require or disallow trailing commas",
		Options: OptionsSchema{
			Type: "string",
			Enum: []string{"never", "always", "always-multiline", "only-multiline"},
		},
	},
	"brace-style": {
		Description: "Enforce consistent brace style for blocks",
		Options: OptionsSchema{
			Type: "string",
			Enum: []string{"1tbs", "stroustrup", "allman"},
		},
	},

	// Imports
	"no-restricted-imports": {
		Description: "Disallow specified modules when loaded by import",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"paths":    {Type: "array", Items: "string"},
				"patterns": {Type: "array", Items: "string"},
			},
		},
	},
	"no-duplicate-imports": {
		Description: "Disallow duplicate module imports",
	},

	// Best Practices
	"curly": {
		Description: "Enforce consistent brace style for all control statements",
		Options: OptionsSchema{
			Type: "string",
			Enum: []string{"all", "multi", "multi-line", "multi-or-nest", "consistent"},
		},
	},
	"dot-notation": {
		Description: "Enforce dot notation whenever possible",
	},
	"no-else-return": {
		Description: "Disallow else blocks after return statements in if statements",
	},
	"no-empty": {
		Description: "Disallow empty block statements",
	},
	"no-empty-function": {
		Description: "Disallow empty functions",
	},
	"no-magic-numbers": {
		Description: "Disallow magic numbers",
		Options: OptionsSchema{
			Type: "object",
			Properties: map[string]OptionProperty{
				"ignore":              {Type: "array", Items: "number"},
				"ignoreArrayIndexes":  {Type: "boolean"},
				"ignoreDefaultValues": {Type: "boolean"},
				"enforceConst":        {Type: "boolean"},
				"detectObjects":       {Type: "boolean"},
			},
		},
	},
	"no-throw-literal": {
		Description: "Disallow throwing literals as exceptions",
	},
	"no-useless-return": {
		Description: "Disallow redundant return statements",
	},
	"require-await": {
		Description: "Disallow async functions which have no await expression",
	},
}

// PrettierOptionRegistry contains all valid Prettier options
var PrettierOptionRegistry = map[string]OptionProperty{
	"printWidth":                {Type: "number", Default: 80},
	"tabWidth":                  {Type: "number", Default: 2},
	"useTabs":                   {Type: "boolean", Default: false},
	"semi":                      {Type: "boolean", Default: true},
	"singleQuote":               {Type: "boolean", Default: false},
	"quoteProps":                {Type: "string", Enum: []string{"as-needed", "consistent", "preserve"}},
	"jsxSingleQuote":            {Type: "boolean", Default: false},
	"trailingComma":             {Type: "string", Enum: []string{"all", "es5", "none"}},
	"bracketSpacing":            {Type: "boolean", Default: true},
	"bracketSameLine":           {Type: "boolean", Default: false},
	"arrowParens":               {Type: "string", Enum: []string{"always", "avoid"}},
	"proseWrap":                 {Type: "string", Enum: []string{"always", "never", "preserve"}},
	"htmlWhitespaceSensitivity": {Type: "string", Enum: []string{"css", "strict", "ignore"}},
	"endOfLine":                 {Type: "string", Enum: []string{"lf", "crlf", "cr", "auto"}},
	"singleAttributePerLine":    {Type: "boolean", Default: false},
}

// TSCOptionRegistry contains all valid TypeScript compiler options for linting
var TSCOptionRegistry = map[string]OptionProperty{
	// Strict Checks
	"strict":                       {Type: "boolean", Default: false},
	"noImplicitAny":                {Type: "boolean", Default: false},
	"strictNullChecks":             {Type: "boolean", Default: false},
	"strictFunctionTypes":          {Type: "boolean", Default: false},
	"strictBindCallApply":          {Type: "boolean", Default: false},
	"strictPropertyInitialization": {Type: "boolean", Default: false},
	"noImplicitThis":               {Type: "boolean", Default: false},
	"useUnknownInCatchVariables":   {Type: "boolean", Default: false},
	"alwaysStrict":                 {Type: "boolean", Default: false},

	// Linting
	"noUnusedLocals":                     {Type: "boolean", Default: false},
	"noUnusedParameters":                 {Type: "boolean", Default: false},
	"exactOptionalPropertyTypes":         {Type: "boolean", Default: false},
	"noImplicitReturns":                  {Type: "boolean", Default: false},
	"noFallthroughCasesInSwitch":         {Type: "boolean", Default: false},
	"noUncheckedIndexedAccess":           {Type: "boolean", Default: false},
	"noImplicitOverride":                 {Type: "boolean", Default: false},
	"noPropertyAccessFromIndexSignature": {Type: "boolean", Default: false},
	"allowUnusedLabels":                  {Type: "boolean", Default: true},
	"allowUnreachableCode":               {Type: "boolean", Default: true},
}

// RuleDefinition defines a linter rule's schema
type RuleDefinition struct {
	Description string
	Options     OptionsSchema
	Deprecated  bool
	Replacement string // If deprecated, which rule replaces it
}

// OptionsSchema defines the schema for rule options
type OptionsSchema struct {
	Type       string                    // "object", "string", "number", "boolean", "array", "mixed"
	Properties map[string]OptionProperty // For object type
	Items      string                    // For array type, element type
	Enum       []string                  // Valid values for string type
}

// OptionProperty defines a single option property
type OptionProperty struct {
	Type    string
	Enum    []string
	Items   string      // For arrays
	Default interface{} // Default value
}

// ValidateESLintRule checks if a rule name and options are valid
func ValidateESLintRule(ruleName string, options interface{}) ValidationError {
	def, exists := ESLintRuleRegistry[ruleName]
	if !exists {
		return ValidationError{
			Valid:   false,
			Message: "unknown ESLint rule: " + ruleName,
			Suggestion: "This rule may require a plugin or doesn't exist. " +
				"Consider using llm-validator for this check instead.",
		}
	}

	if def.Deprecated {
		return ValidationError{
			Valid:      false,
			Message:    "deprecated rule: " + ruleName,
			Suggestion: "Use '" + def.Replacement + "' instead.",
		}
	}

	// TODO: Add options validation based on OptionsSchema
	return ValidationError{Valid: true}
}

// ValidatePrettierOption checks if a Prettier option is valid
func ValidatePrettierOption(optionName string, value interface{}) ValidationError {
	def, exists := PrettierOptionRegistry[optionName]
	if !exists {
		return ValidationError{
			Valid:   false,
			Message: "unknown Prettier option: " + optionName,
		}
	}

	// Validate type and enum if applicable
	if len(def.Enum) > 0 {
		strVal, ok := value.(string)
		if ok {
			valid := false
			for _, allowed := range def.Enum {
				if strVal == allowed {
					valid = true
					break
				}
			}
			if !valid {
				return ValidationError{
					Valid:      false,
					Message:    "invalid value for " + optionName,
					Suggestion: "Valid values: " + joinStrings(def.Enum),
				}
			}
		}
	}

	return ValidationError{Valid: true}
}

// ValidateTSCOption checks if a TypeScript compiler option is valid
func ValidateTSCOption(optionName string, value interface{}) ValidationError {
	_, exists := TSCOptionRegistry[optionName]
	if !exists {
		return ValidationError{
			Valid:   false,
			Message: "unknown TypeScript compiler option: " + optionName,
		}
	}

	return ValidationError{Valid: true}
}

// ValidationError represents a validation result
type ValidationError struct {
	Valid      bool
	Message    string
	Suggestion string
}

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

// GetESLintRuleNames returns all valid ESLint rule names
func GetESLintRuleNames() []string {
	names := make([]string, 0, len(ESLintRuleRegistry))
	for name := range ESLintRuleRegistry {
		names = append(names, name)
	}
	return names
}

// GetPrettierOptionNames returns all valid Prettier option names
func GetPrettierOptionNames() []string {
	names := make([]string, 0, len(PrettierOptionRegistry))
	for name := range PrettierOptionRegistry {
		names = append(names, name)
	}
	return names
}

// GetTSCOptionNames returns all valid TypeScript compiler option names
func GetTSCOptionNames() []string {
	names := make([]string, 0, len(TSCOptionRegistry))
	for name := range TSCOptionRegistry {
		names = append(names, name)
	}
	return names
}
