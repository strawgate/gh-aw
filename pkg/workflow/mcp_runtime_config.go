package workflow

import (
	"fmt"
	"strings"
)

const (
	mcpFromRepoKey         = "from-repo"
	defaultMCPFromRepoPath = "mcp.json"
)

// extractMCPFromRepoConfig extracts reserved runtime config from mcp-servers and
// returns a filtered copy without reserved keys.
//
// Supported syntax:
//
//	mcp-servers:
//	  from-repo:
//	    enabled: true
//	    path: ".github/mcp.json"
func extractMCPFromRepoConfig(mcpServers map[string]any) (*MCPFromRepoConfig, map[string]any, error) {
	filtered := make(map[string]any)
	for key, value := range mcpServers {
		filtered[key] = value
	}

	rawConfig, hasConfig := filtered[mcpFromRepoKey]
	if !hasConfig {
		return nil, filtered, nil
	}

	delete(filtered, mcpFromRepoKey)

	configMap, ok := rawConfig.(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf(
			"mcp-servers.%s must be an object. Example:\n"+
				"mcp-servers:\n"+
				"  from-repo:\n"+
				"    enabled: true\n"+
				"    path: \".github/mcp.json\"",
			mcpFromRepoKey,
		)
	}

	result := &MCPFromRepoConfig{
		Enabled: true,
		Path:    defaultMCPFromRepoPath,
	}

	if enabledAny, exists := configMap["enabled"]; exists {
		enabled, ok := enabledAny.(bool)
		if !ok {
			return nil, nil, fmt.Errorf("mcp-servers.%s.enabled must be a boolean", mcpFromRepoKey)
		}
		result.Enabled = enabled
	}

	if pathAny, exists := configMap["path"]; exists {
		path, ok := pathAny.(string)
		if !ok {
			return nil, nil, fmt.Errorf("mcp-servers.%s.path must be a string", mcpFromRepoKey)
		}
		path = strings.TrimSpace(path)
		if path == "" {
			return nil, nil, fmt.Errorf("mcp-servers.%s.path cannot be empty", mcpFromRepoKey)
		}
		if strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
			return nil, nil, fmt.Errorf(
				"mcp-servers.%s.path must be a repository-relative path without '..' segments, got: %q",
				mcpFromRepoKey,
				path,
			)
		}
		result.Path = path
	}

	// Treat disabled config as absent to keep downstream code simple.
	if !result.Enabled {
		return nil, filtered, nil
	}

	return result, filtered, nil
}
