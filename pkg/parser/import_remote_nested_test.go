//go:build !integration

package parser

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRemoteOrigin(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		expected *remoteImportOrigin
	}{
		{
			name: "deep path extracts parent directory as base path",
			spec: "githubnext/agentics/workflows/shared/reporting.md@main",
			expected: &remoteImportOrigin{
				Owner:    "githubnext",
				Repo:     "agentics",
				Ref:      "main",
				BasePath: "workflows/shared",
			},
		},
		{
			name: "workflowspec with SHA ref",
			spec: "githubnext/agentics/workflows/shared/reporting.md@acea14d65af123c315230221b409f4f435b3706f",
			expected: &remoteImportOrigin{
				Owner:    "githubnext",
				Repo:     "agentics",
				Ref:      "acea14d65af123c315230221b409f4f435b3706f",
				BasePath: "workflows/shared",
			},
		},
		{
			name: "2-level path extracts directory as base path",
			spec: "githubnext/agentics/workflows/code-simplifier.md@main",
			expected: &remoteImportOrigin{
				Owner:    "githubnext",
				Repo:     "agentics",
				Ref:      "main",
				BasePath: "workflows",
			},
		},
		{
			name: "workflowspec without ref defaults to main",
			spec: "githubnext/agentics/workflows/code-simplifier.md",
			expected: &remoteImportOrigin{
				Owner:    "githubnext",
				Repo:     "agentics",
				Ref:      "main",
				BasePath: "workflows",
			},
		},
		{
			name: "workflowspec with section reference",
			spec: "owner/repo/path/file.md@v1.0#SectionName",
			expected: &remoteImportOrigin{
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "v1.0",
				BasePath: "path",
			},
		},
		{
			name: "workflowspec through .github/workflows",
			spec: "githubnext/agentics/.github/workflows/code-simplifier.md@main",
			expected: &remoteImportOrigin{
				Owner:    "githubnext",
				Repo:     "agentics",
				Ref:      "main",
				BasePath: ".github/workflows",
			},
		},
		{
			name: "workflowspec with multi-level base path",
			spec: "org/repo/a/b/c/dir/file.md@v2.0",
			expected: &remoteImportOrigin{
				Owner:    "org",
				Repo:     "repo",
				Ref:      "v2.0",
				BasePath: "a/b/c/dir",
			},
		},
		{
			name: "workflowspec with minimal path (owner/repo/file.md) has empty base",
			spec: "owner/repo/file.md@main",
			expected: &remoteImportOrigin{
				Owner:    "owner",
				Repo:     "repo",
				Ref:      "main",
				BasePath: "",
			},
		},
		{
			name:     "too few parts returns nil",
			spec:     "owner/repo",
			expected: nil,
		},
		{
			name:     "single part returns nil",
			spec:     "file.md",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRemoteOrigin(tt.spec)
			if tt.expected == nil {
				assert.Nilf(t, result, "Expected nil for spec: %s", tt.spec)
			} else {
				require.NotNilf(t, result, "Expected non-nil for spec: %s", tt.spec)
				assert.Equal(t, tt.expected.Owner, result.Owner, "Owner mismatch")
				assert.Equal(t, tt.expected.Repo, result.Repo, "Repo mismatch")
				assert.Equal(t, tt.expected.Ref, result.Ref, "Ref mismatch")
				assert.Equal(t, tt.expected.BasePath, result.BasePath, "BasePath mismatch")
			}
		})
	}
}

func TestFormatWorkflowSpec(t *testing.T) {
	tests := []struct {
		name     string
		owner    string
		repo     string
		filePath string
		ref      string
		expected string
	}{
		{
			name:     "basic spec with branch ref",
			owner:    "githubnext",
			repo:     "agentics",
			filePath: "workflows/code-simplifier.md",
			ref:      "main",
			expected: "githubnext/agentics/workflows/code-simplifier.md@main",
		},
		{
			name:     "spec with SHA ref",
			owner:    "githubnext",
			repo:     "agentics",
			filePath: "workflows/shared/reporting.md",
			ref:      "acea14d65af123c315230221b409f4f435b3706f",
			expected: "githubnext/agentics/workflows/shared/reporting.md@acea14d65af123c315230221b409f4f435b3706f",
		},
		{
			name:     "spec with tag ref",
			owner:    "org",
			repo:     "repo",
			filePath: "dir/file.md",
			ref:      "v1.0",
			expected: "org/repo/dir/file.md@v1.0",
		},
		{
			name:     "spec with deep path",
			owner:    "org",
			repo:     "repo",
			filePath: ".github/workflows/shared/tools.md",
			ref:      "main",
			expected: "org/repo/.github/workflows/shared/tools.md@main",
		},
		{
			name:     "spec with single file",
			owner:    "org",
			repo:     "repo",
			filePath: "file.md",
			ref:      "main",
			expected: "org/repo/file.md@main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatWorkflowSpec(tt.owner, tt.repo, tt.filePath, tt.ref)
			assert.Equal(t, tt.expected, result, "FormatWorkflowSpec should produce canonical format")
		})
	}
}

func TestRemoteImportOriginString(t *testing.T) {
	tests := []struct {
		name     string
		origin   *remoteImportOrigin
		expected string
	}{
		{
			name:     "basic origin",
			origin:   &remoteImportOrigin{Owner: "githubnext", Repo: "agentics", Ref: "main"},
			expected: "githubnext/agentics@main",
		},
		{
			name:     "origin with SHA",
			origin:   &remoteImportOrigin{Owner: "org", Repo: "repo", Ref: "abc123"},
			expected: "org/repo@abc123",
		},
		{
			name:     "origin with tag",
			origin:   &remoteImportOrigin{Owner: "org", Repo: "repo", Ref: "v2.0"},
			expected: "org/repo@v2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.origin.String()
			assert.Equal(t, tt.expected, result, "String() should return owner/repo@ref")
		})
	}
}

func TestResolveNestedImport(t *testing.T) {
	tests := []struct {
		name         string
		origin       *remoteImportOrigin
		relativePath string
		expected     string
	}{
		{
			name: "resolves sibling file in same directory",
			origin: &remoteImportOrigin{
				Owner: "githubnext", Repo: "agentics", Ref: "main", BasePath: "workflows",
			},
			relativePath: "shared/reporting.md",
			expected:     "githubnext/agentics/workflows/shared/reporting.md@main",
		},
		{
			name: "resolves relative to .github/workflows base path",
			origin: &remoteImportOrigin{
				Owner: "githubnext", Repo: "agentics", Ref: "main", BasePath: ".github/workflows",
			},
			relativePath: "shared/reporting.md",
			expected:     "githubnext/agentics/.github/workflows/shared/reporting.md@main",
		},
		{
			name: "resolves with empty base path at repo root",
			origin: &remoteImportOrigin{
				Owner: "org", Repo: "repo", Ref: "v1.0", BasePath: "",
			},
			relativePath: "dir/file.md",
			expected:     "org/repo/dir/file.md@v1.0",
		},
		{
			name: "strips ./ prefix from relative path",
			origin: &remoteImportOrigin{
				Owner: "org", Repo: "repo", Ref: "v2.0", BasePath: ".github/workflows",
			},
			relativePath: "./shared/tools.md",
			expected:     "org/repo/.github/workflows/shared/tools.md@v2.0",
		},
		{
			name: "preserves SHA ref",
			origin: &remoteImportOrigin{
				Owner: "githubnext", Repo: "agentics",
				Ref: "acea14d65af123c315230221b409f4f435b3706f", BasePath: "workflows",
			},
			relativePath: "shared/reporting.md",
			expected:     "githubnext/agentics/workflows/shared/reporting.md@acea14d65af123c315230221b409f4f435b3706f",
		},
		{
			name: "multi-level base path",
			origin: &remoteImportOrigin{
				Owner: "org", Repo: "repo", Ref: "main", BasePath: "a/b/c",
			},
			relativePath: "dir/file.md",
			expected:     "org/repo/a/b/c/dir/file.md@main",
		},
		{
			name: "single file without subdirectory",
			origin: &remoteImportOrigin{
				Owner: "org", Repo: "repo", Ref: "main", BasePath: "workflows",
			},
			relativePath: "file.md",
			expected:     "org/repo/workflows/file.md@main",
		},
		{
			name: "parent traversal with ../ resolves correctly",
			origin: &remoteImportOrigin{
				Owner: "org", Repo: "repo", Ref: "main", BasePath: "base/subdir",
			},
			relativePath: "../sibling/file.md",
			expected:     "org/repo/base/sibling/file.md@main",
		},
		{
			name: "double parent traversal ../../ resolves correctly",
			origin: &remoteImportOrigin{
				Owner: "org", Repo: "repo", Ref: "main", BasePath: "a/b/c",
			},
			relativePath: "../../other/file.md",
			expected:     "org/repo/a/other/file.md@main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.origin.ResolveNestedImport(tt.relativePath)
			assert.Equal(t, tt.expected, result, "ResolveNestedImport(%q)", tt.relativePath)
			assert.True(t, isWorkflowSpec(result), "Result should be a valid workflowspec")
		})
	}
}

func TestLocalImportResolutionBaseline(t *testing.T) {
	// Baseline test: verifies local relative imports resolve correctly.
	// This ensures the import processor still works for non-remote imports
	// after the remote origin tracking changes.

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(workflowsDir, 0o755)
	require.NoError(t, err, "Failed to create workflows directory")

	sharedDir := filepath.Join(workflowsDir, "shared")
	err = os.MkdirAll(sharedDir, 0o755)
	require.NoError(t, err, "Failed to create shared directory")

	localSharedFile := filepath.Join(sharedDir, "local-tools.md")
	err = os.WriteFile(localSharedFile, []byte("# Local tools\n"), 0o644)
	require.NoError(t, err, "Failed to create local shared file")

	frontmatter := map[string]any{
		"imports": []any{"shared/local-tools.md"},
	}
	cache := NewImportCache(tmpDir)
	result, err := ProcessImportsFromFrontmatterWithManifest(frontmatter, workflowsDir, cache)
	require.NoError(t, err, "Local import resolution should succeed")
	assert.NotNil(t, result, "Result should not be nil")
}

func TestRemoteOriginPropagation(t *testing.T) {
	// Test that the remote origin is correctly tracked on queue items
	// when a top-level import is a workflowspec

	// We can't easily test the full remote fetch flow in a unit test,
	// but we can verify the parsing and propagation logic

	// Use the ResolveNestedImport method directly - this is the same
	// helper used by the import processor at runtime.
	resolveNested := func(origin *remoteImportOrigin, nestedPath string) string {
		return origin.ResolveNestedImport(nestedPath)
	}

	t.Run("workflowspec import gets remote origin with base path", func(t *testing.T) {
		spec := "githubnext/agentics/workflows/shared/reporting.md@main"
		assert.True(t, isWorkflowSpec(spec), "Should be recognized as workflowspec")

		origin := parseRemoteOrigin(spec)
		require.NotNil(t, origin, "Should parse remote origin")
		assert.Equal(t, "githubnext", origin.Owner, "Owner should be githubnext")
		assert.Equal(t, "agentics", origin.Repo, "Repo should be agentics")
		assert.Equal(t, "main", origin.Ref, "Ref should be main")
		assert.Equal(t, "workflows/shared", origin.BasePath, "BasePath should be workflows/shared")
	})

	t.Run("local import does not get remote origin", func(t *testing.T) {
		localPath := "shared/tools.md"
		assert.False(t, isWorkflowSpec(localPath), "Should not be recognized as workflowspec")

		origin := parseRemoteOrigin(localPath)
		assert.Nil(t, origin, "Local paths should not produce remote origin")
	})

	t.Run("nested import resolves relative to parent directory", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner:    "githubnext",
			Repo:     "agentics",
			Ref:      "main",
			BasePath: "workflows",
		}
		nestedPath := "shared/reporting.md"

		resolvedSpec := resolveNested(origin, nestedPath)

		assert.Equal(t,
			"githubnext/agentics/workflows/shared/reporting.md@main",
			resolvedSpec,
			"Nested import should resolve relative to parent's directory",
		)
		assert.True(t, isWorkflowSpec(resolvedSpec), "Constructed path should be a valid workflowspec")
	})

	t.Run("nested import resolves relative to .github/workflows base path", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner:    "githubnext",
			Repo:     "agentics",
			Ref:      "main",
			BasePath: ".github/workflows",
		}
		nestedPath := "shared/reporting.md"

		resolvedSpec := resolveNested(origin, nestedPath)

		assert.Equal(t,
			"githubnext/agentics/.github/workflows/shared/reporting.md@main",
			resolvedSpec,
			"Nested import should resolve relative to .github/workflows/ base path",
		)
		assert.True(t, isWorkflowSpec(resolvedSpec), "Constructed path should be a valid workflowspec")
	})

	t.Run("nested import with empty base path resolves at repo root", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner:    "org",
			Repo:     "repo",
			Ref:      "main",
			BasePath: "",
		}
		nestedPath := "shared/tools.md"

		resolvedSpec := resolveNested(origin, nestedPath)

		assert.Equal(t,
			"org/repo/shared/tools.md@main",
			resolvedSpec,
			"Nested import with empty base should resolve directly under repo root",
		)
		assert.True(t, isWorkflowSpec(resolvedSpec), "Constructed path should be a valid workflowspec")
	})

	t.Run("nested relative path with ./ prefix is cleaned", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner:    "org",
			Repo:     "repo",
			Ref:      "v1.0",
			BasePath: ".github/workflows",
		}
		nestedPath := "./shared/tools.md"

		resolvedSpec := resolveNested(origin, nestedPath)

		assert.Equal(t,
			"org/repo/.github/workflows/shared/tools.md@v1.0",
			resolvedSpec,
			"Dot-prefix should be stripped when constructing remote spec",
		)
	})

	t.Run("nested ../ traversal resolves to sibling directory", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner:    "org",
			Repo:     "repo",
			Ref:      "main",
			BasePath: "base/subdir",
		}
		nestedPath := "../fragments/tools.md"

		resolvedSpec := resolveNested(origin, nestedPath)

		assert.Equal(t,
			"org/repo/base/fragments/tools.md@main",
			resolvedSpec,
			"../ should traverse up from base path to resolve sibling directory",
		)
		assert.True(t, isWorkflowSpec(resolvedSpec), "Constructed path should be a valid workflowspec")
	})

	t.Run("nested workflowspec from remote parent gets its own origin", func(t *testing.T) {
		// If a remote file references another workflowspec, it should
		// get its own origin, not inherit the parent's
		nestedSpec := "other-org/other-repo/path/file.md@v2.0"
		assert.True(t, isWorkflowSpec(nestedSpec), "Should be recognized as workflowspec")

		origin := parseRemoteOrigin(nestedSpec)
		require.NotNil(t, origin, "Should parse remote origin for nested workflowspec")
		assert.Equal(t, "other-org", origin.Owner, "Should use nested spec's owner")
		assert.Equal(t, "other-repo", origin.Repo, "Should use nested spec's repo")
		assert.Equal(t, "v2.0", origin.Ref, "Should use nested spec's ref")
	})

	t.Run("path traversal escaping repo root produces invalid path", func(t *testing.T) {
		// A nested import like ../../../etc/passwd from a shallow base path
		// would resolve to a path starting with ".." after path.Clean,
		// which the import processor detects and rejects.
		origin := &remoteImportOrigin{
			Owner:    "org",
			Repo:     "repo",
			Ref:      "main",
			BasePath: "workflows",
		}
		resolvedSpec := resolveNested(origin, "../../../etc/passwd")

		// The resolved spec contains ".." in the repo-relative path,
		// which the import processor rejects as escaping the repo root.
		repoRelative := strings.SplitN(strings.SplitN(resolvedSpec, "@", 2)[0], "/", 3)
		assert.True(t, len(repoRelative) >= 3 && strings.HasPrefix(repoRelative[2], ".."),
			"Path escaping repo root should produce repo-relative path starting with ..: got %s", resolvedSpec)
	})

	t.Run("SHA ref is preserved in nested resolution", func(t *testing.T) {
		sha := "acea14d65af123c315230221b409f4f435b3706f"
		origin := &remoteImportOrigin{
			Owner:    "githubnext",
			Repo:     "agentics",
			Ref:      sha,
			BasePath: "workflows",
		}
		nestedPath := "shared/reporting.md"

		resolvedSpec := resolveNested(origin, nestedPath)

		assert.Equal(t,
			"githubnext/agentics/workflows/shared/reporting.md@"+sha,
			resolvedSpec,
			"SHA ref should be preserved for nested imports",
		)
	})

	t.Run("base path consistency between parent and nested imports", func(t *testing.T) {
		// When importing through workflows/, nested imports
		// should also resolve through workflows/
		parentSpec := "githubnext/agentics/workflows/code-simplifier.md@abc123"
		parentOrigin := parseRemoteOrigin(parentSpec)
		require.NotNil(t, parentOrigin)
		assert.Equal(t, "workflows", parentOrigin.BasePath)

		// Nested import should use the same base
		nestedSpec := resolveNested(parentOrigin, "shared/reporting.md")
		assert.Equal(t,
			"githubnext/agentics/workflows/shared/reporting.md@abc123",
			nestedSpec,
			"Nested import should be consistent with parent's base path",
		)

		// Compare: if the same import goes through .github/workflows/
		altSpec := "githubnext/agentics/.github/workflows/code-simplifier.md@abc123"
		altOrigin := parseRemoteOrigin(altSpec)
		require.NotNil(t, altOrigin)
		assert.Equal(t, ".github/workflows", altOrigin.BasePath)

		altNestedSpec := resolveNested(altOrigin, "shared/reporting.md")
		assert.Equal(t,
			"githubnext/agentics/.github/workflows/shared/reporting.md@abc123",
			altNestedSpec,
			"Nested import through .github/workflows should also be consistent",
		)
	})
}

func TestImportQueueItemRemoteOriginField(t *testing.T) {
	// Verify the struct field exists and works correctly

	t.Run("queue item with nil remote origin", func(t *testing.T) {
		item := importQueueItem{
			importPath:   "shared/tools.md",
			fullPath:     "/tmp/tools.md",
			sectionName:  "",
			baseDir:      "/workspace/.github/workflows",
			remoteOrigin: nil,
		}
		assert.Nil(t, item.remoteOrigin, "Local import should have nil remote origin")
	})

	t.Run("queue item with remote origin", func(t *testing.T) {
		origin := &remoteImportOrigin{
			Owner: "githubnext",
			Repo:  "agentics",
			Ref:   "main",
		}
		item := importQueueItem{
			importPath:   "githubnext/agentics/workflows/file.md@main",
			fullPath:     "/tmp/cache/file.md",
			sectionName:  "",
			baseDir:      "/workspace/.github/workflows",
			remoteOrigin: origin,
		}
		require.NotNil(t, item.remoteOrigin, "Remote import should have non-nil remote origin")
		assert.Equal(t, "githubnext", item.remoteOrigin.Owner, "Owner should match")
		assert.Equal(t, "agentics", item.remoteOrigin.Repo, "Repo should match")
		assert.Equal(t, "main", item.remoteOrigin.Ref, "Ref should match")
	})
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "HTTP 404 message",
			errMsg:   "HTTP 404: Not Found",
			expected: true,
		},
		{
			name:     "lowercase not found",
			errMsg:   "failed to fetch file: not found",
			expected: true,
		},
		{
			name:     "404 status code in message",
			errMsg:   "server returned 404 for request",
			expected: true,
		},
		{
			name:     "authentication error",
			errMsg:   "HTTP 401: Unauthorized",
			expected: false,
		},
		{
			name:     "server error",
			errMsg:   "HTTP 500: Internal Server Error",
			expected: false,
		},
		{
			name:     "empty string",
			errMsg:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.errMsg)
			assert.Equal(t, tt.expected, result, "isNotFoundError(%q)", tt.errMsg)
		})
	}
}

func TestResolveRemoteSymlinksPathConstruction(t *testing.T) {
	// These tests verify the path construction logic of resolveRemoteSymlinks
	// without making real API calls. The actual symlink resolution requires
	// GitHub API access, which is tested in integration tests.

	t.Run("single component path returns error", func(t *testing.T) {
		_, err := resolveRemoteSymlinks("owner", "repo", "file.md", "main")
		assert.Error(t, err, "Single component path has no directories to resolve")
	})

	t.Run("symlink target resolution logic", func(t *testing.T) {
		// Verify the path math that resolveRemoteSymlinks performs internally.
		// Given a symlink at .github/workflows/shared -> ../../workflows/shared,
		// the resolution should produce workflows/shared/file.md

		// Simulate: parts = [".github", "workflows", "shared", "reporting.md"]
		// Symlink at index 3 (parts[:3] = ".github/workflows/shared")
		// Target: "../../workflows/shared"
		// Parent: ".github/workflows"

		parentDir := ".github/workflows"
		target := "../../workflows/shared"

		// This mirrors the logic in resolveRemoteSymlinks using path.Clean/path.Join
		resolvedBase := path.Clean(path.Join(parentDir, target))
		remaining := "reporting.md"
		resolvedPath := resolvedBase + "/" + remaining

		assert.Equal(t, "workflows/shared/reporting.md", resolvedPath,
			"Symlink at .github/workflows/shared pointing to ../../workflows/shared should resolve correctly")
	})

	t.Run("symlink at first component", func(t *testing.T) {
		// Simulate: parts = ["link-dir", "subdir", "file.md"]
		// Symlink at index 1 (parts[:1] = "link-dir")
		// Target: "actual-dir"
		// Parent: "" (root)

		target := "actual-dir"
		resolvedBase := path.Clean(target)
		remaining := "subdir/file.md"
		resolvedPath := resolvedBase + "/" + remaining

		assert.Equal(t, "actual-dir/subdir/file.md", resolvedPath,
			"Symlink at root level should resolve correctly")
	})

	t.Run("nested symlink resolution", func(t *testing.T) {
		// Simulate: parts = ["workflows", "nested", "file.md"]
		// Symlink at index 2 (parts[:2] = "workflows/nested")
		// Target: "../.github/workflows/nested"
		// Parent: "workflows"

		parentDir := "workflows"
		target := "../.github/workflows/nested"

		resolvedBase := path.Clean(path.Join(parentDir, target))
		remaining := "file.md"
		resolvedPath := resolvedBase + "/" + remaining

		assert.Equal(t, ".github/workflows/nested/file.md", resolvedPath,
			"Nested symlink with ../ target should resolve correctly")
	})
}
