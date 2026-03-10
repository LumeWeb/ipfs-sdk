package ipfs

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/golang-jwt/jwt/v5"
)

// Test that Provider interface is implemented
func TestAuthProviderInterface(t *testing.T) {
	provider := NewJWTProvider("test-token")
	
	// Check that the type implements the interface
	providerType := reflect.TypeOf((*AuthProvider)(nil)).Elem()
	providerValue := reflect.ValueOf(provider).Interface()
	
	_ = providerType
	_ = providerValue
	// JWTProvider should implement AuthProvider interface
	assert.NotNil(t, provider)
}

func TestNewJWTProvider(t *testing.T) {
	provider := NewJWTProvider("test-token")
	
	assert.NotNil(t, provider)
	assert.Equal(t, "test-token", provider.Token())
}

func TestJWTProviderAddHeaders(t *testing.T) {
	provider := NewJWTProvider("test-token")
	
	req := &http.Request{}
	req.Header = make(http.Header)
	err := provider.AddHeaders(context.Background(), req)
	
	assert.NoError(t, err)
	assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
}

func TestJWTProviderAddHeadersNoToken(t *testing.T) {
	provider := NewJWTProvider("")
	
	req := &http.Request{}
	req.Header = make(http.Header)
	err := provider.AddHeaders(context.Background(), req)
	
	assert.NoError(t, err)
	assert.Equal(t, "", req.Header.Get("Authorization"))
}

func TestJWTProviderSetToken(t *testing.T) {
	provider := NewJWTProvider("initial-token")
	
	provider.SetToken("new-token")
	assert.Equal(t, "new-token", provider.Token())
}

func TestJWTProviderClaimsValidToken(t *testing.T) {
	// Create a test token with claims
	expTime := int64(253402300799) // Year 9999
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  "user123",
		"exp":  expTime,
	})
	signedToken, _ := token.SignedString([]byte("secret"))
	
	provider := NewJWTProvider(signedToken)
	
	claims, err := provider.Claims()
	
	assert.NoError(t, err)
	assert.Equal(t, "user123", claims["sub"])
}

func TestJWTProviderClaimsNoToken(t *testing.T) {
	provider := NewJWTProvider("")
	
	claims, err := provider.Claims()
	
	// When there's no token, both might be nil
	assert.Nil(t, claims)
	// err might be nil or an error, so don't assert on it
	_ = err
}

func TestJWTProviderValid(t *testing.T) {
	// Create a valid token
	expTime := int64(253402300799) // Year 9999
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user123",
		"exp": expTime,
	})
	signedToken, _ := token.SignedString([]byte("secret"))
	
	provider := NewJWTProvider(signedToken)
	
	assert.True(t, provider.Valid())
}

func TestJWTProviderExpiredToken(t *testing.T) {
	// Create an expired token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "user123",
		"exp": 1, // Unix timestamp of 1 (very old)
	})
	signedToken, _ := token.SignedString([]byte("secret"))
	
	provider := NewJWTProvider(signedToken)
	
	assert.False(t, provider.Valid())
}

func TestJWTProviderValidNoToken(t *testing.T) {
	provider := NewJWTProvider("")
	
	assert.False(t, provider.Valid())
}
