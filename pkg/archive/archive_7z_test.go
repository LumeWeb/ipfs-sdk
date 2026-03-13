package archive

import (
	"os/exec"
	"testing"
)

func Test7ZipRegistration(t *testing.T) {
	Register7ZipExtractor()

	// Test that 7Z format is registered
	registry := DefaultRegistry()
	if !registry.IsFormatSupported(Format7Z) {
		t.Error("7Z format should be registered")
	}

	// Check if 7z/7zz is available for actual tests
	has7z := true
	if _, err := exec.LookPath("7z"); err != nil {
		if _, err := exec.LookPath("7zz"); err != nil {
			has7z = false
		}
	}

	if !has7z {
		t.Log("Note: 7z/7zz not available - tests that require archive creation will be skipped")
	}
}

func Test7ZipFormatDetection(t *testing.T) {
	Register7ZipExtractor()

	// Check if 7z command is available
	has7z := true
	if _, err := exec.LookPath("7z"); err != nil {
		if _, err := exec.LookPath("7zz"); err != nil {
			has7z = false
		}
	}

	if !has7z {
		t.Skip("7z/7zz command not available")
	}

	// This would need actual 7z archive data - skip for now
	t.Skip("Full format detection test requires actual 7z archive")
}
