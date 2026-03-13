package archive

import (
	"os/exec"
	"testing"
)

func TestRarRegistration(t *testing.T) {
	RegisterRarExtractor()

	// Test that RAR format is registered
	registry := DefaultRegistry()
	if !registry.IsFormatSupported(FormatRAR) {
		t.Error("RAR format should be registered")
	}

	// Check if rar is available for actual tests
	hasRar := false
	if _, err := exec.LookPath("rar"); err == nil {
		hasRar = true
	}

	if !hasRar {
		t.Log("Note: rar not available - tests that require archive creation will be skipped")
	}
}

func TestRarFormatDetection(t *testing.T) {
	RegisterRarExtractor()

	// Check if rar command is available
	if _, err := exec.LookPath("rar"); err != nil {
		t.Skip("rar command not available")
	}

	// This would need actual RAR archive data - skip for now
	t.Skip("Full format detection test requires actual RAR archive")
}
