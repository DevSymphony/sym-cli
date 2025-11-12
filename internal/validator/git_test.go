package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractAddedLines_EmptyDiff(t *testing.T) {
	diff := ""
	lines := ExtractAddedLines(diff)
	assert.Empty(t, lines)
}

func TestExtractAddedLines_NoAdditions(t *testing.T) {
	diff := `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -1,3 +1,2 @@
 package main
-import "fmt"
 func main() {}`

	lines := ExtractAddedLines(diff)
	assert.Empty(t, lines)
}

func TestExtractAddedLines_WithAdditions(t *testing.T) {
	diff := `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -1,2 +1,4 @@
 package main
+import "fmt"
+
 func main() {
+	fmt.Println("hello")
 }`

	lines := ExtractAddedLines(diff)

	assert.Len(t, lines, 3)
	assert.Contains(t, lines, `import "fmt"`)
	assert.Contains(t, lines, ``)
	assert.Contains(t, lines, `	fmt.Println("hello")`)
}

func TestExtractAddedLines_IgnoreDiffHeaders(t *testing.T) {
	diff := `diff --git a/test.go b/test.go
index 1234..5678
+++ b/test.go
@@ -1,2 +1,3 @@
+new line`

	lines := ExtractAddedLines(diff)

	// Should only include the actual added line, not the +++ header
	assert.Len(t, lines, 1)
	assert.Equal(t, "new line", lines[0])
}
