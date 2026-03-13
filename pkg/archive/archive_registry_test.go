package archive

import (
	"bytes"
	"io"
	"testing"

	"github.com/mholt/archives"
)

func TestArchiveRegistry(t *testing.T) {
	// Register extractors for this test
	RegisterZipExtractor()
	RegisterTarExtractor()

	// Use the default registry
	registry := DefaultRegistry()

	tests := []struct {
		name   string
		format Format
	}{
		{"ZIP", FormatZIP},
		{"TAR", FormatTAR},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !registry.IsFormatSupported(tt.format) {
				t.Errorf("%s format should be supported", tt.name)
			}
		})
	}

	// Test supported formats
	formats := registry.SupportedFormats()
	if len(formats) == 0 {
		t.Error("At least one format should be supported")
	}

	// Verify all expected formats are in supported formats
	expectedFormats := map[Format]bool{
		FormatZIP: false,
		FormatTAR: false,
	}

	for _, format := range formats {
		if _, exists := expectedFormats[format]; exists {
			expectedFormats[format] = true
		}
	}

	for format, found := range expectedFormats {
		if !found {
			t.Errorf("Format %s should be in supported formats", format.String())
		}
	}
}

func TestUnsupportedFormat(t *testing.T) {
	registry := NewArchiveRegistry() // Empty registry, no detectors

	// Test with a reader that doesn't support seeking
	reader := bytes.NewReader([]byte{0x00, 0x01, 0x02, 0x03})

	// Create a non-seeker reader by wrapping it
	nonSeekerReader := io.NopCloser(reader)

	_, err := registry.DetectFormat(nonSeekerReader)
	if err == nil {
		t.Error("Should have failed to detect format with non-seeking reader")
	}

	// Test with seeking reader but no registered extractors
	_, err = registry.DetectFormat(reader)
	if err == nil {
		t.Error("Should have failed to detect format with no registered extractors")
	}
}

func TestFormatFileDetection(t *testing.T) {
	registry := NewArchiveRegistry()
	RegisterDefaultDetectors(registry) // Register detectors but no extractors

	// Test with regular file content (text)
	fileContent := []byte("This is a regular text file content")
	reader := bytes.NewReader(fileContent)

	format, err := registry.DetectFormat(reader)
	if err != nil {
		t.Errorf("Should detect FormatFile successfully, got error: %v", err)
	}
	if format != FormatFile {
		t.Errorf("Expected FormatFile, got %s", format.String())
	}
}

func TestRegistryThreadSafety(t *testing.T) {
	registry := NewArchiveRegistry()
	RegisterDefaultDetectors(registry)

	// Test concurrent access
	done := make(chan bool, 2)

	// Goroutine 1: Check supported formats
	go func() {
		registry.SupportedFormats()
		done <- true
	}()

	// Goroutine 2: Check format support
	go func() {
		registry.IsFormatSupported(FormatZIP)
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done
}

func TestRegistryWithMocks(t *testing.T) {
	registry := NewArchiveRegistry()
	RegisterDefaultDetectors(registry)

	// Test with mock extractor using factory
	mockExtractor := NewMockArchiveExtractor(t)
	mockExtractor.EXPECT().Format().Return(FormatZIP)
	mockExtractor.EXPECT().Close().Return(nil)

	mockExtractorFactory := func(reader archives.ReaderAtSeeker) (ArchiveExtractor, error) {
		return mockExtractor, nil
	}

	// Register mock extractor
	registry.RegisterExtractor(FormatZIP, mockExtractorFactory)

	// Test that format is supported
	if !registry.IsFormatSupported(FormatZIP) {
		t.Error("ZIP format should be supported")
	}

	// Test supported formats includes ZIP
	formats := registry.SupportedFormats()
	found := false
	for _, format := range formats {
		if format == FormatZIP {
			found = true
			break
		}
	}
	if !found {
		t.Error("ZIP format should be in supported formats")
	}

	// Actually create an extractor to satisfy mock expectations
	testData := bytes.NewReader([]byte("test data"))
	extractor, err := registry.CreateExtractorForFormat(FormatZIP, testData)
	if err != nil {
		t.Errorf("Failed to get extractor: %v", err)
	}
	if extractor == nil {
		t.Error("Extractor should not be nil")
	}

	// Call the methods that the mock expects
	if extractor.Format() != FormatZIP {
		t.Error("Format should return FormatZIP")
	}

	if err := extractor.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestFormatIsArchiveFormat(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		want   bool
	}{
		{
			name:   "UNKNOWN format is not archive",
			format: FormatUnknown,
			want:   false,
		},
		{
			name:   "CAR format is not archive",
			format: FormatCAR,
			want:   false,
		},
		{
			name:   "FILE format is not archive",
			format: FormatFile,
			want:   false,
		},
		{
			name:   "ZIP format is archive",
			format: FormatZIP,
			want:   true,
		},
		{
			name:   "RAR format is archive",
			format: FormatRAR,
			want:   true,
		},
		{
			name:   "TAR format is archive",
			format: FormatTAR,
			want:   true,
		},
		{
			name:   "TAR.GZ format is archive",
			format: FormatTAR_GZ,
			want:   true,
		},
		{
			name:   "TAR.BZ2 format is archive",
			format: FormatTAR_BZ2,
			want:   true,
		},
		{
			name:   "7Z format is archive",
			format: Format7Z,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format.IsArchiveFormat(); got != tt.want {
				t.Errorf("Format.IsArchiveFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatIsUploadFormat(t *testing.T) {
	tests := []struct {
		name   string
		format Format
		want   bool
	}{
		{
			name:   "UNKNOWN format is not upload",
			format: FormatUnknown,
			want:   false,
		},
		{
			name:   "CAR format is upload",
			format: FormatCAR,
			want:   true,
		},
		{
			name:   "FILE format is not upload",
			format: FormatFile,
			want:   false,
		},
		{
			name:   "ZIP format is not upload",
			format: FormatZIP,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format.IsUploadFormat(); got != tt.want {
				t.Errorf("Format.IsUploadFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}
