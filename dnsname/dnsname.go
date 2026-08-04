// Package dnsname provides shared, dependency-free helpers for normalizing
// and comparing DNS domain names across the trailing-dot ambiguity.
//
// DNS names may be written as a bare/relative name ("lumeweb") or as a
// properly-terminated FQDN ("lumeweb."). The terminating dot is the DNS root
// separator (not an empty label), so both forms are canonically equivalent.
//
// All dot-manipulation functions in this package are CASE-PRESERVING. DNS is
// case-insensitive for comparisons, but some consumers (HNS on-chain names,
// DNS record content) must retain the original case. Only Equal folds case,
// for comparisons. Callers that need lowercase should call strings.ToLower
// themselves.
//
// The empty string "" is preserved by all functions. The DNS root zone "." is
// a distinct case handled by callers that need it; it is not special-cased
// here.
package dnsname

import "strings"

// TrimDot removes a single trailing root dot ("" → "").
// "lumeweb." → "lumeweb"; "lumeweb" → "lumeweb". Case-preserving.
func TrimDot(s string) string {
	return strings.TrimSuffix(s, ".")
}

// Normalize returns the canonical bare form (no trailing dot). Case-PRESERVING.
// "Lumeweb." → "Lumeweb"; "lumeweb" → "lumeweb".
func Normalize(s string) string {
	return TrimDot(s)
}

// Equal reports whether two DNS names are equivalent regardless of trailing
// dot or case. Equal("lumeweb.", "Lumeweb") == true. The DNS root zone "." is
// distinct from the empty string: Equal(".", "") == false.
func Equal(a, b string) bool {
	// Keep "." (root zone) and "" distinct as documented.
	if a == "." || b == "." {
		return a == b
	}
	return strings.EqualFold(TrimDot(a), TrimDot(b))
}

// IsFQDN reports whether s ends in the terminating dot (and is non-empty).
func IsFQDN(s string) bool {
	return s != "" && strings.HasSuffix(s, ".")
}

// EnsureFQDN appends a trailing dot if not already present. Case-preserving.
// "lumeweb" → "lumeweb."; "lumeweb." → "lumeweb."; "" → "".
func EnsureFQDN(s string) string {
	if s == "" || strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

// Canonical returns the fully-qualified (trailing-dot) form. Case-PRESERVING.
// "lumeweb" → "lumeweb."; "lumeweb." → "lumeweb."; "" → "".
func Canonical(s string) string {
	return EnsureFQDN(s)
}
