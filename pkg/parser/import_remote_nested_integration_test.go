//go:build integration

package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipOnAuthErr skips the test when GitHub API authentication is unavailable.
func skipOnAuthErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	if strings.Contains(msg, "auth") || strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "authentication token not found") {
		t.Skip("Skipping: GitHub API authentication not available")
	}
}

// TestRemoteNestedImportResolution verifies the end-to-end flow of importing
// a remote workflow whose nested relative imports must resolve through the
// parent's base path.
//
// Uses the githubnext/agentics repository which has this structure:
//
//	workflows/
//	  code-simplifier.md       (imports shared/reporting.md)
//	  shared/
//	    reporting.md            (fragment with no further imports)
//
// When code-simplifier.md is imported via workflowspec, its nested import
// "shared/reporting.md" must resolve to workflows/shared/reporting.md using
// the parent file's directory (workflows/) as the base path.
func TestRemoteNestedImportResolution(t *testing.T) {
	spec := "githubnext/agentics/workflows/code-simplifier.md@main"

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0o755), "Failed to create workflows directory")

	cache := NewImportCache(tmpDir)

	frontmatter := map[string]any{
		"imports": []any{spec},
	}

	result, err := ProcessImportsFromFrontmatterWithManifest(frontmatter, workflowsDir, cache)
	if err != nil {
		skipOnAuthErr(t, err)
		t.Fatalf("ProcessImportsFromFrontmatterWithManifest failed: %v", err)
	}
	require.NotNil(t, result, "Result should not be nil")

	// --- Verify both the parent and nested import appear in ImportedFiles ---
	t.Run("imported files contain parent and nested imports", func(t *testing.T) {
		require.NotEmpty(t, result.ImportedFiles, "ImportedFiles should not be empty")

		foundParent := false
		foundNested := false
		for _, f := range result.ImportedFiles {
			if strings.Contains(f, "code-simplifier.md") {
				foundParent = true
			}
			if strings.Contains(f, "reporting.md") {
				foundNested = true
			}
		}
		assert.True(t, foundParent, "code-simplifier.md should be in ImportedFiles")
		assert.True(t, foundNested, "reporting.md (nested import) should be in ImportedFiles")
	})

	// --- Verify ImportedFiles entries are full workflowspecs ---
	t.Run("imported files are full workflowspecs", func(t *testing.T) {
		for _, importedFile := range result.ImportedFiles {
			assert.True(t, isWorkflowSpec(importedFile),
				"Import should be a full workflowspec, got: %s", importedFile)
			assert.Contains(t, importedFile, "githubnext/agentics/",
				"Workflowspec should include owner/repo prefix: %s", importedFile)
			assert.Contains(t, importedFile, "@",
				"Workflowspec should include @ref suffix: %s", importedFile)
		}
	})

	// --- Verify nested import resolved through correct base path ---
	t.Run("nested import resolves through workflows base path", func(t *testing.T) {
		found := false
		for _, importedFile := range result.ImportedFiles {
			if strings.Contains(importedFile, "reporting.md") {
				found = true
				assert.Contains(t, importedFile, "workflows/shared/reporting.md",
					"Nested import should resolve through workflows/ directory: %s", importedFile)
			}
		}
		assert.True(t, found, "Should have found reporting.md in ImportedFiles")
	})

	// --- Verify cache entries exist with correct path prefixes ---
	t.Run("remote files are cached with correct prefixes", func(t *testing.T) {
		fullCacheDir := cache.GetCacheDir()

		var cachedFiles []string
		err := filepath.Walk(fullCacheDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
				cachedFiles = append(cachedFiles, info.Name())
			}
			return nil
		})
		require.NoError(t, err, "Walking cache directory should not error")
		require.NotEmpty(t, cachedFiles, "Should have cached .md files")
		t.Logf("Cached files: %v", cachedFiles)

		foundReporting := false
		for _, name := range cachedFiles {
			if strings.Contains(name, "reporting") {
				foundReporting = true
				assert.Contains(t, name, "workflows_shared_reporting.md",
					"Cached reporting.md should have workflows_shared_ prefix: %s", name)
			}
		}
		assert.True(t, foundReporting, "reporting.md should be in cache")
	})

	// --- Verify ImportPaths for runtime-import macro generation ---
	t.Run("import paths reference cached files", func(t *testing.T) {
		require.NotEmpty(t, result.ImportPaths, "ImportPaths should not be empty")

		for _, importPath := range result.ImportPaths {
			assert.Contains(t, importPath, "imports/githubnext/agentics",
				"Import path should reference cache directory: %s", importPath)
		}
	})
}

// TestRemoteImportCacheReuse verifies that processing the same remote
// workflowspec twice reuses cached content without re-downloading, and
// produces identical results.
func TestRemoteImportCacheReuse(t *testing.T) {
	spec := "githubnext/agentics/workflows/shared/reporting.md@main"

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0o755))

	cache := NewImportCache(tmpDir)

	frontmatter := map[string]any{
		"imports": []any{spec},
	}

	// First pass - downloads from GitHub
	result1, err := ProcessImportsFromFrontmatterWithManifest(frontmatter, workflowsDir, cache)
	if err != nil {
		skipOnAuthErr(t, err)
		t.Fatalf("First import pass failed: %v", err)
	}
	require.NotNil(t, result1)

	// Second pass - should use cache
	result2, err := ProcessImportsFromFrontmatterWithManifest(frontmatter, workflowsDir, cache)
	require.NoError(t, err, "Second import pass (cached) should succeed")
	require.NotNil(t, result2)

	// Both passes should produce the same number of ImportedFiles
	assert.Equal(t, len(result1.ImportedFiles), len(result2.ImportedFiles),
		"Cached pass should produce same number of imported files")

	// Both should produce the same number of ImportPaths
	assert.Equal(t, len(result1.ImportPaths), len(result2.ImportPaths),
		"Cached pass should produce same number of import paths")
}

// TestRemoteImportOriginParsing verifies that parseRemoteOrigin correctly
// extracts fields from real workflowspec paths in the githubnext/agentics
// repository, including BasePath extraction at different path depths.
func TestRemoteImportOriginParsing(t *testing.T) {
	tests := []struct {
		name             string
		spec             string
		expectedOwner    string
		expectedRepo     string
		expectedRef      string
		expectedBasePath string
	}{
		{
			name:             "file in workflows directory has workflows base path",
			spec:             "githubnext/agentics/workflows/code-simplifier.md@main",
			expectedOwner:    "githubnext",
			expectedRepo:     "agentics",
			expectedRef:      "main",
			expectedBasePath: "workflows",
		},
		{
			name:             "file in workflows/shared has deeper base path",
			spec:             "githubnext/agentics/workflows/shared/reporting.md@main",
			expectedOwner:    "githubnext",
			expectedRepo:     "agentics",
			expectedRef:      "main",
			expectedBasePath: "workflows/shared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origin := parseRemoteOrigin(tt.spec)
			require.NotNil(t, origin, "Should parse remote origin for spec: %s", tt.spec)
			assert.Equal(t, tt.expectedOwner, origin.Owner, "Owner mismatch")
			assert.Equal(t, tt.expectedRepo, origin.Repo, "Repo mismatch")
			assert.Equal(t, tt.expectedRef, origin.Ref, "Ref mismatch")
			assert.Equal(t, tt.expectedBasePath, origin.BasePath, "BasePath mismatch")
		})
	}
}

// TestRemoteImportResolveIncludePath verifies that ResolveIncludePath
// correctly downloads and caches files from the githubnext/agentics
// repository when given a full workflowspec.
func TestRemoteImportResolveIncludePath(t *testing.T) {
	spec := "githubnext/agentics/workflows/shared/reporting.md@main"

	tmpDir := t.TempDir()
	cache := NewImportCache(tmpDir)

	resolvedPath, err := ResolveIncludePath(spec, "", cache)
	if err != nil {
		skipOnAuthErr(t, err)
		t.Fatalf("ResolveIncludePath failed: %v", err)
	}

	assert.NotEmpty(t, resolvedPath, "Resolved path should not be empty")
	assert.Contains(t, resolvedPath, "imports/githubnext/agentics",
		"Path should be in cache directory")

	// Verify the file exists and has content
	content, err := os.ReadFile(resolvedPath)
	require.NoError(t, err, "Should be able to read cached file")
	assert.NotEmpty(t, content, "Cached file should have content")
	assert.Contains(t, string(content), "Report", "Content should be a reporting fragment")
}
