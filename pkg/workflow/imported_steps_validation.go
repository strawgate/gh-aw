// This file provides validation for imported steps in engine configurations.
//
// # Imported Steps Validation
//
// This file previously validated that custom engine steps did not use agentic engine
// secrets. With the removal of custom engine support, this validation is now a no-op.
// The file is kept for backwards compatibility with the compiler orchestrator.
//
// For general validation, see validation.go.
// For strict mode validation, see strict_mode_validation.go.
// For detailed documentation, see scratchpad/validation-architecture.md

package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var importedStepsValidationLog = logger.New("workflow:imported_steps_validation")

// validateImportedStepsNoAgenticSecrets validates that engine steps don't use agentic engine secrets
// This validation is now a no-op since custom engine support has been removed.
// The function is kept for backwards compatibility but always returns nil.
func (c *Compiler) validateImportedStepsNoAgenticSecrets(engineConfig *EngineConfig, engineID string) error {
	// Custom engine has been removed, so this validation no longer applies
	importedStepsValidationLog.Print("Skipping validation: custom engine support removed")
	return nil
}
