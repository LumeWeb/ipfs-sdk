package dnsname

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrimDot(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty stays empty", "", ""},
		{"bare name", "lumeweb", "lumeweb"},
		{"single trailing dot", "lumeweb.", "lumeweb"},
		{"mixed case trailing dot", "Lumeweb.", "Lumeweb"},
		{"multi-label", "a.b.c", "a.b.c"},
		{"multi-label trailing dot", "a.b.c.", "a.b.c"},
		{"fqdn with trailing dot", "ns1.example.com.", "ns1.example.com"},
		{"just a dot", ".", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, TrimDot(tt.in))
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty stays empty", "", ""},
		{"bare name unchanged", "lumeweb", "lumeweb"},
		{"trailing dot stripped", "lumeweb.", "lumeweb"},
		{"mixed case PRESERVED (no lowercase forcing)", "Lumeweb.", "Lumeweb"},
		{"uppercase preserved", "EXAMPLE.COM.", "EXAMPLE.COM"},
		{"multi-label", "a.b.c.", "a.b.c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Normalize(tt.in))
		})
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{"both bare equal", "lumeweb", "lumeweb", true},
		{"trailing dot vs bare", "lumeweb.", "lumeweb", true},
		{"bare vs trailing dot", "lumeweb", "lumeweb.", true},
		{"both trailing dot", "lumeweb.", "lumeweb.", true},
		{"case insensitive", "Lumeweb.", "lumeweb", true},
		{"case insensitive reverse", "lumeweb", "LUMEWEB.", true},
		{"multi-label case insensitive", "NS1.EXAMPLE.COM.", "ns1.example.com", true},
		{"different names", "lumeweb", "lumeweb2", false},
		{"subdomain not equal", "ns1.lumeweb", "lumeweb", false},
		{"empty vs empty", "", "", true},
		{"empty vs non-empty", "", "lumeweb", false},
		{"one bare one empty", "lumeweb", "", false},
		{"root zone vs empty are distinct", ".", "", false},
		{"empty vs root zone are distinct", "", ".", false},
		{"root zone vs root zone equal", ".", ".", true},
		{"root zone vs its fqdn-dot form", ".", "..", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Equal(tt.a, tt.b), "Equal(%q, %q)", tt.a, tt.b)
		})
	}
}

func TestIsFQDN(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty is not fqdn", "", false},
		{"bare name is not fqdn", "lumeweb", false},
		{"trailing dot is fqdn", "lumeweb.", true},
		{"multi-label trailing dot is fqdn", "ns1.example.com.", true},
		{"just dot is fqdn", ".", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsFQDN(tt.in))
		})
	}
}

func TestEnsureFQDN(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty stays empty", "", ""},
		{"bare name gets dot", "lumeweb", "lumeweb."},
		{"already fqdn unchanged", "lumeweb.", "lumeweb."},
		{"case preserved, gets dot", "Lumeweb", "Lumeweb."},
		{"case preserved, already dot", "Lumeweb.", "Lumeweb."},
		{"multi-label", "ns1.example.com", "ns1.example.com."},
		{"just a dot unchanged", ".", "."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, EnsureFQDN(tt.in))
		})
	}
}

func TestCanonical(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty stays empty", "", ""},
		{"bare name gets dot", "lumeweb", "lumeweb."},
		{"already fqdn unchanged", "lumeweb.", "lumeweb."},
		{"case preserved", "Lumeweb", "Lumeweb."},
		{"case preserved already dot", "Lumeweb.", "Lumeweb."},
		{"multi-label", "a.b.c", "a.b.c."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Canonical(tt.in))
		})
	}
}

// TestCanonicalRoundTrip ensures Canonical is idempotent.
func TestCanonicalRoundTrip(t *testing.T) {
	inputs := []string{"", "lumeweb", "lumeweb.", "Lumeweb.", "a.b.c.", "ns1.example.com"}
	for _, in := range inputs {
		once := Canonical(in)
		require.Equal(t, once, Canonical(once), "Canonical should be idempotent for %q", in)
	}
}
