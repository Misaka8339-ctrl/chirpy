package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestJWT(t *testing.T) {
	userID := uuid.New()
	const secret = "test-only-secret-not-for-production"

	tests := []struct {
		name         string
		expiresIn    time.Duration
		verifySecret string
		wantErr      error
	}{
		{
			name:         "valid token",
			expiresIn:    time.Hour,
			verifySecret: secret,
		},
		{
			name:         "expired token",
			expiresIn:    -time.Hour,
			verifySecret: secret,
			wantErr:      jwt.ErrTokenExpired,
		},
		{
			name:         "wrong secret",
			expiresIn:    time.Hour,
			verifySecret: "different-secret",
			wantErr:      jwt.ErrTokenSignatureInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString, err := MakeJWT(userID, secret, tt.expiresIn)
			if err != nil {
				t.Fatalf("MakeJWT failed: %v", err)
			}

			gotID, err := ValidateJWT(tokenString, tt.verifySecret)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				if gotID != uuid.Nil {
					t.Fatalf("expected uuid.Nil on failure, got %v", gotID)
				}
				return
			}

			if err != nil {
				t.Fatalf("ValidateJWT failed: %v", err)
			}
			if gotID != userID {
				t.Fatalf("user ID = %v, want %v", gotID, userID)
			}
		})
	}
}

func TestValidateJWTMalformed(t *testing.T) {
	_, err := ValidateJWT("not-a-jwt", "test-secret")
	if err == nil {
		t.Fatal("expected malformed token to be rejected")
	}
}
