package eslint

import (
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

func TestParseASTQuery(t *testing.T) {
	tests := []struct {
		name    string
		rule    *core.Rule
		want    *ASTQuery
		wantErr bool
	}{
		{
			name: "simple node query",
			rule: &core.Rule{
				Check: map[string]interface{}{
					"node": "FunctionDeclaration",
				},
			},
			want: &ASTQuery{
				Node: "FunctionDeclaration",
			},
			wantErr: false,
		},
		{
			name: "query with where clause",
			rule: &core.Rule{
				Check: map[string]interface{}{
					"node": "FunctionDeclaration",
					"where": map[string]interface{}{
						"async": true,
					},
				},
			},
			want: &ASTQuery{
				Node: "FunctionDeclaration",
				Where: map[string]interface{}{
					"async": true,
				},
			},
			wantErr: false,
		},
		{
			name: "query with has clause",
			rule: &core.Rule{
				Check: map[string]interface{}{
					"node": "FunctionDeclaration",
					"has":  []interface{}{"TryStatement"},
				},
			},
			want: &ASTQuery{
				Node: "FunctionDeclaration",
				Has:  []string{"TryStatement"},
			},
			wantErr: false,
		},
		{
			name: "missing node",
			rule: &core.Rule{
				Check: map[string]interface{}{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseASTQuery(tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseASTQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Node != tt.want.Node {
				t.Errorf("Node = %v, want %v", got.Node, tt.want.Node)
			}
			if len(tt.want.Where) > 0 && got.Where == nil {
				t.Errorf("Where is nil, want %v", tt.want.Where)
			}
			if len(tt.want.Has) > 0 && len(got.Has) != len(tt.want.Has) {
				t.Errorf("Has = %v, want %v", got.Has, tt.want.Has)
			}
		})
	}
}

func TestGenerateESTreeSelector(t *testing.T) {
	tests := []struct {
		name  string
		query *ASTQuery
		want  string
	}{
		{
			name: "simple node",
			query: &ASTQuery{
				Node: "FunctionDeclaration",
			},
			want: "FunctionDeclaration",
		},
		{
			name: "node with where clause",
			query: &ASTQuery{
				Node: "FunctionDeclaration",
				Where: map[string]interface{}{
					"async": true,
				},
			},
			want: "FunctionDeclaration[async=true]",
		},
		{
			name: "node with has clause",
			query: &ASTQuery{
				Node: "FunctionDeclaration",
				Where: map[string]interface{}{
					"async": true,
				},
				Has: []string{"TryStatement"},
			},
			want: "FunctionDeclaration[async=true]:not(:has(TryStatement))",
		},
		{
			name: "node with notHas clause",
			query: &ASTQuery{
				Node: "FunctionDeclaration",
				NotHas: []string{"ReturnStatement"},
			},
			want: "FunctionDeclaration:has(ReturnStatement)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateESTreeSelector(tt.query)
			if got != tt.want {
				t.Errorf("GenerateESTreeSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}
