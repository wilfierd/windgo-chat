package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// GitHubUser represents GitHub user response
type GitHubUserResponse struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubKey represents GitHub SSH key
type GitHubKey struct {
	ID  int    `json:"id"`
	Key string `json:"key"`
}

// GenerateNonce creates a cryptographically secure random nonce
func GenerateNonce() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateDeviceID creates a unique device identifier
func GenerateDeviceID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("dev_%s", base64.URLEncoding.EncodeToString(bytes)[:22]), nil
}

// GetGitHubUser fetches user information from GitHub API
func GetGitHubUser(username string) (*GitHubUserResponse, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s", username)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user GitHubUserResponse
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetGitHubSSHKeys fetches SSH public keys from GitHub API
func GetGitHubSSHKeys(username string) ([]GitHubKey, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/keys", username)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var keys []GitHubKey
	if err := json.Unmarshal(body, &keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// VerifySSHSignature verifies SSH signature against GitHub public keys
func VerifySSHSignature(username, signature, fingerprint, nonce string) (bool, error) {
	// Get GitHub SSH keys
	keys, err := GetGitHubSSHKeys(username)
	if err != nil {
		return false, err
	}

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}

	// Try each key
	for _, key := range keys {
		// Parse SSH public key
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key.Key))
		if err != nil {
			continue
		}

		// Calculate key fingerprint
		keyFingerprint := ssh.FingerprintSHA256(pubKey)
		
		// Check if fingerprint matches
		if keyFingerprint != fingerprint {
			continue
		}

		// Verify signature
		if err := pubKey.Verify([]byte(nonce), &ssh.Signature{
			Format: pubKey.Type(),
			Blob:   sigBytes,
		}); err == nil {
			return true, nil
		}
	}

	return false, fmt.Errorf("signature verification failed")
}

// HashFingerprint creates a SHA256 hash of the fingerprint
func HashFingerprint(fingerprint string) string {
	hash := sha256.Sum256([]byte(fingerprint))
	return base64.URLEncoding.EncodeToString(hash[:])
}

// ValidateDeviceID checks if device ID format is valid
func ValidateDeviceID(deviceID string) bool {
	return strings.HasPrefix(deviceID, "dev_") && len(deviceID) >= 26
}

// GenerateRefreshToken creates a secure refresh token
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// IsNonceExpired checks if a nonce has expired
func IsNonceExpired(createdAt time.Time, ttl time.Duration) bool {
	return time.Now().After(createdAt.Add(ttl))
}
