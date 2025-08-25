package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// NonceResponse represents the nonce API response
type NonceResponse struct {
	NonceID string `json:"nonce_id"`
	Nonce   string `json:"nonce"`
}

// SSHLoginRequest represents the SSH login request
type SSHLoginRequest struct {
	GitHubUser     string `json:"github_user"`
	Signed         string `json:"signed"`
	PubFingerprint string `json:"pub_fingerprint"`
	NonceID        string `json:"nonce_id"`
	DeviceName     string `json:"device_name"`
	DeviceType     string `json:"device_type"`
}

// DeviceResponse represents the login response
type DeviceResponse struct {
	UserID   uint   `json:"user_id"`
	DeviceID string `json:"device_id"`
	User     struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		AuthType string `json:"auth_type"`
	} `json:"user"`
}

const (
	API_BASE_URL = "http://localhost:8080"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ssh-auth-cli <github_username>")
		os.Exit(1)
	}

	githubUser := os.Args[1]
	
	fmt.Printf("🔐 Starting SSH authentication for GitHub user: %s\n", githubUser)

	// Step 1: Get nonce
	fmt.Print("📡 Getting nonce from server... ")
	nonce, err := getNonce()
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Success (ID: %s)\n", nonce.NonceID[:8]+"...")

	// Step 2: Find SSH key
	fmt.Print("🔑 Finding SSH private key... ")
	privateKeyPath, publicKey, err := findSSHKey()
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Found: %s\n", privateKeyPath)

	// Step 3: Sign nonce
	fmt.Print("✍️  Signing nonce... ")
	signature, fingerprint, err := signNonce(nonce.Nonce, privateKeyPath, publicKey)
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Signed with key: %s\n", fingerprint)

	// Step 4: Login
	fmt.Print("🚀 Authenticating with server... ")
	response, err := sshLogin(SSHLoginRequest{
		GitHubUser:     githubUser,
		Signed:         signature,
		PubFingerprint: fingerprint,
		NonceID:        nonce.NonceID,
		DeviceName:     getDeviceName(),
		DeviceType:     "cli",
	})
	if err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Success!\n")
	fmt.Printf("\n🎉 Welcome %s!\n", response.User.Username)
	fmt.Printf("   User ID: %d\n", response.UserID)
	fmt.Printf("   Device ID: %s\n", response.DeviceID)
	fmt.Printf("   Email: %s\n", response.User.Email)
	fmt.Printf("   Role: %s\n", response.User.Role)
}

func getNonce() (*NonceResponse, error) {
	resp, err := http.Post(API_BASE_URL+"/auth/nonce", "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var nonce NonceResponse
	if err := json.NewDecoder(resp.Body).Decode(&nonce); err != nil {
		return nil, err
	}

	return &nonce, nil
}

func findSSHKey() (string, ssh.PublicKey, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", nil, err
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	
	// Common SSH key names
	keyNames := []string{"id_rsa", "id_ed25519", "id_ecdsa"}
	
	for _, keyName := range keyNames {
		privateKeyPath := filepath.Join(sshDir, keyName)
		publicKeyPath := privateKeyPath + ".pub"
		
		// Check if both private and public keys exist
		if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
			continue
		}
		if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
			continue
		}

		// Read public key
		pubKeyBytes, err := os.ReadFile(publicKeyPath)
		if err != nil {
			continue
		}

		// Parse public key
		publicKey, _, _, _, err := ssh.ParseAuthorizedKey(pubKeyBytes)
		if err != nil {
			continue
		}

		return privateKeyPath, publicKey, nil
	}

	return "", nil, fmt.Errorf("no SSH key found in %s", sshDir)
}

func signNonce(nonce, privateKeyPath string, publicKey ssh.PublicKey) (string, string, error) {
	// Get fingerprint
	fingerprint := ssh.FingerprintSHA256(publicKey)

	// Use ssh-keygen to sign (works with ssh-agent)
	cmd := exec.Command("ssh-keygen", "-Y", "sign", "-f", privateKeyPath, "-n", "windgo-chat")
	cmd.Stdin = strings.NewReader(nonce)
	
	output, err := cmd.Output()
	if err != nil {
		// Fallback: try to read private key directly
		return signNonceDirectly(nonce, privateKeyPath, fingerprint)
	}

	// Extract signature from ssh-keygen output
	lines := strings.Split(string(output), "\n")
	var sigLines []string
	inSig := false
	
	for _, line := range lines {
		if strings.HasPrefix(line, "-----BEGIN SSH SIGNATURE-----") {
			inSig = true
			continue
		}
		if strings.HasPrefix(line, "-----END SSH SIGNATURE-----") {
			break
		}
		if inSig && line != "" {
			sigLines = append(sigLines, line)
		}
	}

	if len(sigLines) == 0 {
		return "", "", fmt.Errorf("failed to extract signature")
	}

	signature := strings.Join(sigLines, "")
	return signature, fingerprint, nil
}

func signNonceDirectly(nonce, privateKeyPath, fingerprint string) (string, string, error) {
	// Read private key
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", "", err
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return "", "", err
	}

	// Sign the nonce
	signature, err := signer.Sign(rand.Reader, []byte(nonce))
	if err != nil {
		return "", "", err
	}

	// Encode signature
	sigBytes := signature.Blob
	encodedSig := base64.StdEncoding.EncodeToString(sigBytes)

	return encodedSig, fingerprint, nil
}

func sshLogin(req SSHLoginRequest) (*DeviceResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(API_BASE_URL+"/auth/login/ssh", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	var response DeviceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func getDeviceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "Unknown CLI"
	}
	return fmt.Sprintf("CLI on %s", hostname)
}
