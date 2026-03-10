package ipfs

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthProvider defines the interface for authentication providers.
// Implementations can add authentication headers and tokens to HTTP requests.
type AuthProvider interface {
	// AddHeaders adds authentication headers to the HTTP request.
	// This is called for each outbound request to apply authentication.
	AddHeaders(ctx context.Context, req *http.Request) error
	
	// Token returns the current authentication token.
	// Returns an empty string if no token is set.
	Token() string
	
	// SetToken sets the authentication token.
	SetToken(token string)
}

// JWTProvider is a JWT-based authentication provider.
// It automatically adds "Bearer {token}" authorization headers to requests.
type JWTProvider struct {
	token string
}

// NewJWTProvider creates a new JWT authentication provider.
// The token parameter is the JWT token string.
func NewJWTProvider(token string) *JWTProvider {
	return &JWTProvider{token: token}
}

// AddHeaders adds the authorization header with the JWT token.
// If no token is set, no header is added.
func (p *JWTProvider) AddHeaders(_ context.Context, req *http.Request) error {
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	return nil
}

// Token returns the current JWT token.
func (p *JWTProvider) Token() string {
	return p.token
}

// SetToken sets a new JWT token.
// This can be used to refresh tokens during runtime.
func (p *JWTProvider) SetToken(token string) {
	p.token = token
}

// Claims retrieves the JWT claims from the current token.
// Returns an error if the token is not valid or cannot be parsed.
func (p *JWTProvider) Claims() (jwt.MapClaims, error) {
	if p.token == "" {
		return nil, nil
	}

	parser := jwt.NewParser()
	claims := jwt.MapClaims{}

	// Parse without verification for claim inspection
	_, _, err := parser.ParseUnverified(p.token, &claims)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

// Valid checks if the current token is valid.
// This performs basic validation without full cryptographic verification.
func (p *JWTProvider) Valid() bool {
	if p.token == "" {
		return false
	}

	claims, err := p.Claims()
	if err != nil {
		return false
	}

	// Check if expired
	if exp, ok := claims["exp"].(float64); ok {
		currentTime := float64(time.Now().Unix())
		if exp < currentTime {
			return false
		}
	}

	return true
}
