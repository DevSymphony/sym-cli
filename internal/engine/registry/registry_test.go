package registry

import (
	"context"
	"testing"

	"github.com/DevSymphony/sym-cli/internal/engine/core"
)

// mockEngine is a mock implementation for testing
type mockEngine struct {
	name string
}

func (m *mockEngine) Init(ctx context.Context, config core.EngineConfig) error {
	return nil
}

func (m *mockEngine) Validate(ctx context.Context, rule core.Rule, files []string) (*core.ValidationResult, error) {
	return &core.ValidationResult{
		RuleID: rule.ID,
		Passed: true,
		Engine: m.name,
	}, nil
}

func (m *mockEngine) GetCapabilities() core.EngineCapabilities {
	return core.EngineCapabilities{
		Name: m.name,
	}
}

func (m *mockEngine) Close() error {
	return nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := &Registry{
		engines:   make(map[string]core.Engine),
		factories: make(map[string]EngineFactory),
	}

	// Register factory
	err := r.Register("test", func() (core.Engine, error) {
		return &mockEngine{name: "test"}, nil
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get engine (should create it)
	engine, err := r.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if engine == nil {
		t.Fatal("Get returned nil engine")
	}

	// Get again (should return same instance)
	engine2, err := r.Get("test")
	if err != nil {
		t.Fatalf("Get (2nd) failed: %v", err)
	}

	if engine != engine2 {
		t.Error("Get returned different instances (should be same)")
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	r := &Registry{
		engines:   make(map[string]core.Engine),
		factories: make(map[string]EngineFactory),
	}

	factory := func() (core.Engine, error) {
		return &mockEngine{name: "test"}, nil
	}

	// First registration should succeed
	if err := r.Register("test", factory); err != nil {
		t.Fatalf("First Register failed: %v", err)
	}

	// Second registration should fail
	err := r.Register("test", factory)
	if err == nil {
		t.Error("Register duplicate succeeded, want error")
	}
}

func TestRegistry_GetNonExistent(t *testing.T) {
	r := &Registry{
		engines:   make(map[string]core.Engine),
		factories: make(map[string]EngineFactory),
	}

	_, err := r.Get("nonexistent")
	if err == nil {
		t.Error("Get(nonexistent) succeeded, want error")
	}
}

func TestRegistry_List(t *testing.T) {
	r := &Registry{
		engines:   make(map[string]core.Engine),
		factories: make(map[string]EngineFactory),
	}

	names := []string{"pattern", "length", "style"}
	for _, name := range names {
		n := name // capture
		if err := r.Register(n, func() (core.Engine, error) {
			return &mockEngine{name: n}, nil
		}); err != nil {
			t.Fatalf("Register(%s) failed: %v", n, err)
		}
	}

	got := r.List()
	if len(got) != len(names) {
		t.Errorf("List() length = %d, want %d", len(got), len(names))
	}
}
