package core

import (
	"testing"
)

func TestRule_GetString(t *testing.T) {
	rule := &Rule{
		Check: map[string]interface{}{
			"engine": "pattern",
			"target": "identifier",
		},
	}

	if got := rule.GetString("engine"); got != "pattern" {
		t.Errorf("GetString(engine) = %q, want %q", got, "pattern")
	}

	if got := rule.GetString("missing"); got != "" {
		t.Errorf("GetString(missing) = %q, want empty", got)
	}
}

func TestRule_GetInt(t *testing.T) {
	rule := &Rule{
		Check: map[string]interface{}{
			"max":   100,
			"float": 50.5,
		},
	}

	if got := rule.GetInt("max"); got != 100 {
		t.Errorf("GetInt(max) = %d, want 100", got)
	}

	if got := rule.GetInt("float"); got != 50 {
		t.Errorf("GetInt(float) = %d, want 50", got)
	}

	if got := rule.GetInt("missing"); got != 0 {
		t.Errorf("GetInt(missing) = %d, want 0", got)
	}
}

func TestRule_GetBool(t *testing.T) {
	rule := &Rule{
		Check: map[string]interface{}{
			"enabled": true,
		},
	}

	if got := rule.GetBool("enabled"); got != true {
		t.Errorf("GetBool(enabled) = %v, want true", got)
	}

	if got := rule.GetBool("missing"); got != false {
		t.Errorf("GetBool(missing) = %v, want false", got)
	}
}

func TestRule_GetStringSlice(t *testing.T) {
	rule := &Rule{
		Check: map[string]interface{}{
			"languages": []interface{}{"javascript", "typescript"},
			"native":    []string{"go", "rust"},
		},
	}

	got := rule.GetStringSlice("languages")
	want := []string{"javascript", "typescript"}
	if len(got) != len(want) {
		t.Fatalf("GetStringSlice(languages) length = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("GetStringSlice(languages)[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if got := rule.GetStringSlice("missing"); got != nil {
		t.Errorf("GetStringSlice(missing) = %v, want nil", got)
	}
}

func TestViolation_String(t *testing.T) {
	tests := []struct {
		name string
		v    Violation
		want string
	}{
		{
			name: "full location",
			v: Violation{
				File:    "src/app.js",
				Line:    10,
				Column:  5,
				Message: "Missing semicolon",
				RuleID:  "STYLE-SEMI",
			},
			want: "src/app.js:10:5: Missing semicolon [STYLE-SEMI]",
		},
		{
			name: "line only",
			v: Violation{
				File:    "src/utils.js",
				Line:    42,
				Message: "Line too long",
				RuleID:  "LENGTH-LINE",
			},
			want: "src/utils.js:42: Line too long [LENGTH-LINE]",
		},
		{
			name: "file only",
			v: Violation{
				File:    "README.md",
				Message: "File too long",
				RuleID:  "LENGTH-FILE",
			},
			want: "README.md: File too long [LENGTH-FILE]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
