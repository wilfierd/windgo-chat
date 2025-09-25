package handlers

import (
    "chat-backend-go/config"
    "chat-backend-go/utils"
    "encoding/json"
    "log"
    "net/http"
    "net/url"
    "strings"
    "time"

    "github.com/gofiber/fiber/v2"
    "golang.org/x/oauth2"
)

type deviceStartResponse struct {
    DeviceCode              string `json:"device_code"`
    UserCode                string `json:"user_code"`
    VerificationURI         string `json:"verification_uri"`
    ExpiresIn               int    `json:"expires_in"`
    Interval                int    `json:"interval"`
    VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
}

type devicePollResponse struct {
    AccessToken string `json:"access_token"`
    TokenType   string `json:"token_type"`
    Scope       string `json:"scope"`
    Error       string `json:"error"`
}

// GitHubDeviceStart starts the device flow and returns user_code and verification URI.
func GitHubDeviceStart(c *fiber.Ctx) error {
    oauthCfg, err := config.GetGitHubOAuthConfig()
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
    }

    form := url.Values{}
    form.Set("client_id", oauthCfg.ClientID)
    form.Set("scope", strings.Join(oauthCfg.Scopes, " "))

    req, _ := http.NewRequest("POST", "https://github.com/login/device/code", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("Accept", "application/json")

    httpClient := &http.Client{Timeout: 10 * time.Second}
    resp, err := httpClient.Do(req)
    if err != nil {
        return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to contact GitHub"})
    }
    defer resp.Body.Close()

    var out deviceStartResponse
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "invalid response from GitHub"})
    }
    if out.DeviceCode == "" || out.UserCode == "" {
        return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to start device flow"})
    }
    return c.JSON(out)
}

// GitHubDevicePoll polls GitHub for token and returns app JWT when authorized.
// Request JSON: { "device_code": string, "timeout": optional seconds, "interval": optional seconds }
func GitHubDevicePoll(c *fiber.Ctx) error {
    var reqBody struct {
        DeviceCode string `json:"device_code"`
        Timeout    int    `json:"timeout"`
        Interval   int    `json:"interval"`
    }
    if err := c.BodyParser(&reqBody); err != nil || reqBody.DeviceCode == "" {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "device_code required"})
    }

    if reqBody.Timeout <= 0 {
        reqBody.Timeout = 60 // default 60s
    }
    if reqBody.Interval <= 0 {
        reqBody.Interval = 5 // default 5s
    }

    oauthCfg, err := config.GetGitHubOAuthConfig()
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
    }

    deadline := time.Now().Add(time.Duration(reqBody.Timeout) * time.Second)
    wait := time.Duration(reqBody.Interval) * time.Second

    httpClient := &http.Client{Timeout: 10 * time.Second}

    for time.Now().Before(deadline) {
        form := url.Values{}
        form.Set("client_id", oauthCfg.ClientID)
        form.Set("device_code", reqBody.DeviceCode)
        form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
        // GitHub requires client_secret for device token exchange
        form.Set("client_secret", oauthCfg.ClientSecret)

        httpReq, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(form.Encode()))
        httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        httpReq.Header.Set("Accept", "application/json")

        resp, err := httpClient.Do(httpReq)
        if err != nil {
            return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to contact GitHub"})
        }

        var pr devicePollResponse
        if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
            resp.Body.Close()
            return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "invalid response from GitHub"})
        }
        resp.Body.Close()

		if pr.AccessToken != "" {
			log.Println("GitHub Device Flow: Received access token, fetching user profile")
			// We have a GitHub user; fetch profile and issue app JWT
			tok := &oauth2.Token{AccessToken: pr.AccessToken}
			ghUser, primaryEmail, err := fetchGitHubUser(tok)
			if err != nil {
				log.Printf("GitHub Device Flow: Error fetching user profile: %v", err)
				return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
			}
			log.Printf("GitHub Device Flow: Fetched user profile, email: %s", primaryEmail)
			user, err := linkOrCreateUserFromGitHub(ghUser, primaryEmail)
			if err != nil {
				log.Printf("GitHub Device Flow: Error creating/linking user: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
			}
			log.Printf("GitHub Device Flow: User processed successfully, ID: %d", user.ID)
			appToken, err := utils.GenerateJWT(user.ID)
			if err != nil {
				log.Printf("GitHub Device Flow: Error generating JWT: %v", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
			}
			log.Printf("GitHub Device Flow: JWT generated successfully for user ID: %d", user.ID)
			return c.JSON(AuthResponse{Token: appToken, User: *user})
		}

        switch pr.Error {
        case "authorization_pending":
            // Continue waiting
        case "slow_down":
            wait += 5 * time.Second
        case "expired_token":
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "device code expired"})
        case "access_denied":
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "access denied"})
        case "":
            // No explicit error but also no access token; wait
        default:
            return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": pr.Error})
        }

        // Sleep until next poll interval or deadline
        select {
        case <-time.After(wait):
        case <-time.After(time.Until(deadline)):
            return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{"error": "timeout"})
        }
    }

    return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{"error": "timeout"})
}
