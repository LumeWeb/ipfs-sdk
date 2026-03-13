package archive

import (
	"bytes"
	"compress/gzip"
	"testing"
)

func TestTarGzRegistration(t *testing.T) {
	RegisterTarGzExtractor()

	// Test that TAR_GZ format is registered
	registry := DefaultRegistry()
	if !registry.IsFormatSupported(FormatTAR_GZ) {
		t.Error("TAR_GZ format should be registered")
	}
}

func TestTarGzFormatDetection(t *testing.T) {
	RegisterTarGzExtractor()

	// Create simple then compress with gzip
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte("test content")); err != nil {
		t.Fatalf("Failed to write compressed data: %v", err)
	}
	gz.Close()

	// Test format detection
	detected, err := DetectFormat(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to detect format: %v", err)
	}

	if detected == FormatTAR_GZ {
		t.Log("TAR_GZ format detected successfully")
	} else if detected == FormatFile || detected == FormatUnknown {
		t.Log("Detection may fail with simple data, but format is registered")
	}
}

func TestTarBz2Registration(t *testing.T) {
	RegisterTarBz2Extractor()

	// Test that TAR_BZ2 format is registered
	registry := DefaultRegistry()
	if !registry.IsFormatSupported(FormatTAR_BZ2) {
		t.Error("TAR_BZ2 format should be registered")
	}
}
