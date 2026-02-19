//go:build !integration

package workflow

import "testing"

func TestExtractMCPFromRepoConfig(t *testing.T) {
	tests := []struct {
		name         string
		mcpServers   map[string]any
		wantErr      bool
		wantConfig   bool
		wantPath     string
		wantFiltered int
	}{
		{
			name: "no runtime config",
			mcpServers: map[string]any{
				"github-extra": map[string]any{"type": "http", "url": "https://example.com/mcp"},
			},
			wantConfig:   false,
			wantPath:     "",
			wantFiltered: 1,
		},
		{
			name: "enabled with defaults",
			mcpServers: map[string]any{
				"from-repo": map[string]any{},
				"github-extra": map[string]any{
					"type": "http",
					"url":  "https://example.com/mcp",
				},
			},
			wantConfig:   true,
			wantPath:     defaultMCPFromRepoPath,
			wantFiltered: 1,
		},
		{
			name: "enabled with explicit path",
			mcpServers: map[string]any{
				"from-repo": map[string]any{
					"enabled": true,
					"path":    ".github/mcp.json",
				},
			},
			wantConfig:   true,
			wantPath:     ".github/mcp.json",
			wantFiltered: 0,
		},
		{
			name: "disabled runtime config",
			mcpServers: map[string]any{
				"from-repo": map[string]any{
					"enabled": false,
				},
			},
			wantConfig:   false,
			wantPath:     "",
			wantFiltered: 0,
		},
		{
			name: "invalid non-object config",
			mcpServers: map[string]any{
				"from-repo": true,
			},
			wantErr: true,
		},
		{
			name: "invalid absolute path",
			mcpServers: map[string]any{
				"from-repo": map[string]any{
					"path": "/tmp/mcp.json",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, filtered, err := extractMCPFromRepoConfig(tt.mcpServers)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := len(filtered); got != tt.wantFiltered {
				t.Fatalf("filtered map length = %d, want %d", got, tt.wantFiltered)
			}
			if _, exists := filtered[mcpFromRepoKey]; exists {
				t.Fatalf("filtered map should not contain reserved key %q", mcpFromRepoKey)
			}

			if tt.wantConfig && config == nil {
				t.Fatalf("expected runtime config, got nil")
			}
			if !tt.wantConfig && config != nil {
				t.Fatalf("expected nil runtime config, got %+v", config)
			}
			if tt.wantConfig && config.Path != tt.wantPath {
				t.Fatalf("config.Path = %q, want %q", config.Path, tt.wantPath)
			}
		})
	}
}
