package config

import (
    "errors"
    "os"

    "golang.org/x/oauth2"
    "golang.org/x/oauth2/github"
)

// GetGitHubOAuthConfig builds an oauth2.Config from environment variables.
// Requires GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET, and GITHUB_REDIRECT_URL.
func GetGitHubOAuthConfig() (*oauth2.Config, error) {
    clientID := os.Getenv("GITHUB_CLIENT_ID")
    clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
    redirectURL := os.Getenv("GITHUB_REDIRECT_URL")

    if clientID == "" || clientSecret == "" || redirectURL == "" {
        return nil, errors.New("GitHub OAuth not configured: set GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET, GITHUB_REDIRECT_URL")
    }

    cfg := &oauth2.Config{
        ClientID:     clientID,
        ClientSecret: clientSecret,
        Endpoint:     github.Endpoint,
        RedirectURL:  redirectURL,
        Scopes:       []string{"read:user", "user:email"},
    }
    return cfg, nil
}

