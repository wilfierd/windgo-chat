package utils

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	DeviceID string `json:"device_id"`
	jwt.RegisteredClaims
}

// getJWTSecret returns the JWT secret from environment or a secure default
func getJWTSecret() string {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Println("Warning: JWT_SECRET not set, using default secret. Set JWT_SECRET environment variable for production!")
		jwtSecret = "windgo-chat-default-secret-please-change-in-production-2024"
	}
	return jwtSecret
}

// GenerateJWT creates a new JWT token for a user and device
func GenerateJWT(userID uint, deviceID string) (string, error) {
	// Get JWT secret
	jwtSecret := getJWTSecret()

	// Create claims
	claims := Claims{
		UserID:   userID,
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Token expires in 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateJWT validates a JWT token and returns the user ID and device ID
func ValidateJWT(tokenString string) (uint, string, error) {
	// Get JWT secret
	jwtSecret := getJWTSecret()

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return 0, "", err
	}

	// Extract claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims.UserID, claims.DeviceID, nil
	}

	return 0, "", errors.New("invalid token")
}

// ExtractUserID extracts user ID from Authorization header
func ExtractUserID(authHeader string) (uint, error) {
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return 0, errors.New("invalid authorization header format")
	}

	tokenString := authHeader[7:]
	userID, _, err := ValidateJWT(tokenString)
	return userID, err
}

// ExtractUserAndDevice extracts user ID and device ID from Authorization header
func ExtractUserAndDevice(authHeader string) (uint, string, error) {
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return 0, "", errors.New("invalid authorization header format")
	}

	tokenString := authHeader[7:]
	return ValidateJWT(tokenString)
}

// RefreshToken generates a new token for an existing user and device
func RefreshToken(userID uint, deviceID string) (string, error) {
	return GenerateJWT(userID, deviceID)
}
