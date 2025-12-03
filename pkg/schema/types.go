package schema

// UserPolicy represents the user-friendly policy schema (A schema)
type UserPolicy struct {
	Version  string        `json:"version,omitempty"`
	RBAC     *UserRBAC     `json:"rbac,omitempty"`
	Defaults *UserDefaults `json:"defaults,omitempty"`
	Rules    []UserRule    `json:"rules"`
}

// UserRBAC represents RBAC configuration in user schema
type UserRBAC struct {
	Roles map[string]UserRole `json:"roles"`
}

// UserRole represents a single role in user schema
type UserRole struct {
	AllowWrite    []string `json:"allowWrite,omitempty"`
	DenyWrite     []string `json:"denyWrite,omitempty"`
	AllowExec     []string `json:"allowExec,omitempty"`
	CanEditPolicy bool     `json:"canEditPolicy,omitempty"` // symphonyclient integration: policy editing permission
	CanEditRoles  bool     `json:"canEditRoles,omitempty"`  // symphonyclient integration: role editing permission
}

// UserDefaults represents default values for rules
type UserDefaults struct {
	Languages       []string `json:"languages,omitempty"`
	DefaultLanguage string   `json:"defaultLanguage,omitempty"` // Default language for new rules
	Include         []string `json:"include,omitempty"`
	Exclude         []string `json:"exclude,omitempty"`
	Severity        string   `json:"severity,omitempty"`
	Autofix         bool     `json:"autofix,omitempty"`
}

// UserRule represents a single rule in user schema
type UserRule struct {
	ID        string         `json:"id"`                   // Rule ID (required, can be number or string)
	Say       string         `json:"say"`
	Category  string         `json:"category,omitempty"`
	Languages []string       `json:"languages,omitempty"`
	Include   []string       `json:"include,omitempty"`
	Exclude   []string       `json:"exclude,omitempty"`
	Severity  string         `json:"severity,omitempty"`
	Autofix   bool           `json:"autofix,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
	Message   string         `json:"message,omitempty"`
	Example   string         `json:"example,omitempty"`
}

// CodePolicy represents the formal validation schema (B schema)
type CodePolicy struct {
	Version string          `json:"version"`
	Project *ProjectInfo    `json:"project,omitempty"`
	Extends []string        `json:"extends,omitempty"`
	RBAC    *PolicyRBAC     `json:"rbac,omitempty"`
	Rules   []PolicyRule    `json:"rules"`
	Enforce EnforceSettings `json:"enforce"`
}

// ProjectInfo represents project metadata
type ProjectInfo struct {
	Name       string   `json:"name,omitempty"`
	Languages  []string `json:"languages,omitempty"`
	Frameworks []string `json:"frameworks,omitempty"`
}

// PolicyRBAC represents RBAC configuration in policy schema
type PolicyRBAC struct {
	Roles map[string]PolicyRole `json:"roles"`
}

// PolicyRole represents a single role in policy schema
type PolicyRole struct {
	Inherits    []string     `json:"inherits,omitempty"`
	Permissions []Permission `json:"permissions"`
}

// Permission represents a single permission entry
type Permission struct {
	Path       string                `json:"path"`
	Read       bool                  `json:"read"`
	Write      bool                  `json:"write"`
	Execute    bool                  `json:"execute"`
	Conditions *PermissionConditions `json:"conditions,omitempty"`
}

// PermissionConditions represents conditions for permissions
type PermissionConditions struct {
	Branches []string   `json:"branches,omitempty"`
	Time     *TimeRange `json:"time,omitempty"`
}

// TimeRange represents time-based conditions
type TimeRange struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

// PolicyRule represents a single rule in policy schema
type PolicyRule struct {
	ID       string         `json:"id"`
	Enabled  bool           `json:"enabled"`
	Category string         `json:"category"`
	Severity string         `json:"severity"`
	Desc     string         `json:"desc,omitempty"`
	When     *Selector      `json:"when,omitempty"`
	Check    map[string]any `json:"check"`
	Remedy   *Remedy        `json:"remedy,omitempty"`
	Message  string         `json:"message,omitempty"`
}

// Selector represents rule application conditions
type Selector struct {
	Languages []string `json:"languages,omitempty"`
	Include   []string `json:"include,omitempty"`
	Exclude   []string `json:"exclude,omitempty"`
	Branches  []string `json:"branches,omitempty"`
	Roles     []string `json:"roles,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// Remedy represents auto-fix configuration
type Remedy struct {
	Autofix bool           `json:"autofix"`
	Tool    string         `json:"tool,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

// EnforceSettings represents enforcement configuration
type EnforceSettings struct {
	Stages     []string     `json:"stages"`
	FailOn     []string     `json:"fail_on,omitempty"`
	RBACConfig *RBACEnforce `json:"rbac,omitempty"`
}

// RBACEnforce represents RBAC enforcement settings
type RBACEnforce struct {
	Enabled     bool     `json:"enabled"`
	Stages      []string `json:"stages,omitempty"`
	OnViolation string   `json:"on_violation,omitempty"`
}
