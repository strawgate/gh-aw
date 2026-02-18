//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveInlinePrompt(t *testing.T) {
	tests := []struct {
		name           string
		compilerFlag   bool
		frontmatter    map[string]any
		expectedResult bool
	}{
		{
			name:           "default is false",
			compilerFlag:   false,
			frontmatter:    map[string]any{},
			expectedResult: false,
		},
		{
			name:           "compiler flag true overrides everything",
			compilerFlag:   true,
			frontmatter:    map[string]any{"inline-prompt": false},
			expectedResult: true,
		},
		{
			name:           "frontmatter true enables inlining",
			compilerFlag:   false,
			frontmatter:    map[string]any{"inline-prompt": true},
			expectedResult: true,
		},
		{
			name:           "frontmatter false keeps default",
			compilerFlag:   false,
			frontmatter:    map[string]any{"inline-prompt": false},
			expectedResult: false,
		},
		{
			name:           "missing frontmatter key uses default",
			compilerFlag:   false,
			frontmatter:    map[string]any{"engine": "copilot"},
			expectedResult: false,
		},
		{
			name:           "non-boolean frontmatter value is ignored",
			compilerFlag:   false,
			frontmatter:    map[string]any{"inline-prompt": "yes"},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Compiler{inlinePrompt: tt.compilerFlag}
			result := c.resolveInlinePrompt(tt.frontmatter)
			assert.Equal(t, tt.expectedResult, result, "resolveInlinePrompt should return expected value")
		})
	}
}

func TestReadImportedMarkdown(t *testing.T) {
	tmpDir := testutil.TempDir(t, "read-import-test")

	fragmentDir := filepath.Join(tmpDir, ".github", "workflows", "shared")
	require.NoError(t, os.MkdirAll(fragmentDir, 0755), "creating fragment directory")

	fragmentContent := `---
description: A shared fragment
---

# Fragment Content

This is the body of the fragment.`
	fragmentPath := filepath.Join(fragmentDir, "fragment.md")
	require.NoError(t, os.WriteFile(fragmentPath, []byte(fragmentContent), 0644), "writing fragment file")

	c := &Compiler{gitRoot: tmpDir}
	markdown, err := c.readImportedMarkdown(".github/workflows/shared/fragment.md")
	require.NoError(t, err, "reading imported markdown should succeed")
	assert.Contains(t, markdown, "# Fragment Content", "should contain the markdown heading")
	assert.Contains(t, markdown, "This is the body of the fragment.", "should contain the body")
	assert.NotContains(t, markdown, "description:", "should not contain frontmatter")
}

func TestReadImportedMarkdown_MissingFile(t *testing.T) {
	tmpDir := testutil.TempDir(t, "read-import-missing")

	c := &Compiler{gitRoot: tmpDir}
	_, err := c.readImportedMarkdown("nonexistent/file.md")
	require.Error(t, err, "should error on missing file")
	assert.Contains(t, err.Error(), "reading import", "error should describe the failure")
}

func TestInlinePromptFromFrontmatter(t *testing.T) {
	tmpDir := testutil.TempDir(t, "inline-prompt-frontmatter")

	sharedDir := filepath.Join(tmpDir, ".github", "workflows", "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0755), "creating shared directory")

	fragmentContent := `# Shared Instructions

Always be helpful.`
	require.NoError(t, os.WriteFile(
		filepath.Join(sharedDir, "instructions.md"),
		[]byte(fragmentContent), 0644),
		"writing shared fragment",
	)

	workflowContent := `---
on: push
inline-prompt: true
strict: false
engine: copilot
imports:
  - shared/instructions.md
features:
  dangerous-permissions-write: true
---

# My Workflow

Do the thing.`
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	workflowFile := filepath.Join(workflowDir, "test-workflow.md")
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0644), "writing workflow file")

	compiler := NewCompiler(WithGitRoot(tmpDir))
	err := compiler.CompileWorkflow(workflowFile)
	require.NoError(t, err, "compilation with inline-prompt should succeed")

	lockFile := filepath.Join(workflowDir, "test-workflow.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "reading lock file")

	content := string(lockContent)
	assert.NotContains(t, content, "{{#runtime-import", "lock file should not contain runtime-import macros")
	assert.Contains(t, content, "Always be helpful", "lock file should contain inlined fragment content")
	assert.Contains(t, content, "Do the thing", "lock file should contain inlined main workflow content")
}

func TestInlinePromptCLIFlag(t *testing.T) {
	tmpDir := testutil.TempDir(t, "inline-prompt-cli")

	sharedDir := filepath.Join(tmpDir, ".github", "workflows", "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0755), "creating shared directory")

	fragmentContent := `# CLI Fragment

Fragment from CLI test.`
	require.NoError(t, os.WriteFile(
		filepath.Join(sharedDir, "cli-fragment.md"),
		[]byte(fragmentContent), 0644),
		"writing fragment",
	)

	workflowContent := `---
on: push
strict: false
engine: copilot
imports:
  - shared/cli-fragment.md
features:
  dangerous-permissions-write: true
---

# CLI Test Workflow

Main body here.`
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	workflowFile := filepath.Join(workflowDir, "cli-test.md")
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0644), "writing workflow")

	compiler := NewCompiler(WithGitRoot(tmpDir), WithInlinePrompt(true))
	err := compiler.CompileWorkflow(workflowFile)
	require.NoError(t, err, "compilation with WithInlinePrompt should succeed")

	lockFile := filepath.Join(workflowDir, "cli-test.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "reading lock file")

	content := string(lockContent)
	assert.NotContains(t, content, "{{#runtime-import", "lock file should not contain runtime-import macros")
	assert.Contains(t, content, "Fragment from CLI test", "should contain inlined fragment")
	assert.Contains(t, content, "Main body here", "should contain inlined main workflow")
}

func TestDefaultCompilationUsesRuntimeImport(t *testing.T) {
	tmpDir := testutil.TempDir(t, "default-runtime-import")

	sharedDir := filepath.Join(tmpDir, ".github", "workflows", "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0755), "creating shared directory")

	fragmentContent := `# Default Fragment

Should be runtime-imported.`
	require.NoError(t, os.WriteFile(
		filepath.Join(sharedDir, "default-fragment.md"),
		[]byte(fragmentContent), 0644),
		"writing fragment",
	)

	workflowContent := `---
on: push
strict: false
engine: copilot
imports:
  - shared/default-fragment.md
features:
  dangerous-permissions-write: true
---

# Default Workflow

Normal compilation.`
	workflowDir := filepath.Join(tmpDir, ".github", "workflows")
	workflowFile := filepath.Join(workflowDir, "default-test.md")
	require.NoError(t, os.WriteFile(workflowFile, []byte(workflowContent), 0644), "writing workflow")

	compiler := NewCompiler(WithGitRoot(tmpDir))
	err := compiler.CompileWorkflow(workflowFile)
	require.NoError(t, err, "default compilation should succeed")

	lockFile := filepath.Join(workflowDir, "default-test.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	require.NoError(t, err, "reading lock file")

	content := string(lockContent)
	runtimeImportCount := strings.Count(content, "{{#runtime-import")
	assert.GreaterOrEqual(t, runtimeImportCount, 2, "default compilation should produce runtime-import macros for fragments and main workflow")
	assert.NotContains(t, content, "Should be runtime-imported", "fragment content should NOT be inlined in default mode")
}
