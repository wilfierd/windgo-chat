package handlers

import (
    "chat-backend-go/config"
    "chat-backend-go/models"
    "chat-backend-go/utils"
    "context"
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/gofiber/fiber/v2"
    "golang.org/x/crypto/bcrypt"
    "golang.org/x/oauth2"
)

const oauthStateCookie = "oauth_state"

func randomState(n int) (string, error) {
    b := make([]byte, n)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.RawURLEncoding.EncodeToString(b), nil
}

// GitHubLogin redirects the user to GitHub's authorization page.
func GitHubLogin(c *fiber.Ctx) error {
    oauthCfg, err := config.GetGitHubOAuthConfig()
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
    }

    state, err := randomState(24)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate state"})
    }

    // Persist state in an HttpOnly cookie with short TTL
    c.Cookie(&fiber.Cookie{
        Name:     oauthStateCookie,
        Value:    state,
        HTTPOnly: true,
        Secure:   strings.HasPrefix(strings.ToLower(c.Protocol()), "https"),
        SameSite: "Lax",
        Expires:  time.Now().Add(10 * time.Minute),
    })

    authURL := oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
    return c.Redirect(authURL, http.StatusFound)
}

// GitHubCallback handles the OAuth callback, exchanges code, and creates/logs in user.
func GitHubCallback(c *fiber.Ctx) error {
    oauthCfg, err := config.GetGitHubOAuthConfig()
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
    }

    state := c.Query("state")
    code := c.Query("code")
    if state == "" || code == "" {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing state or code"})
    }

    // Validate state from cookie
    cookie := c.Cookies(oauthStateCookie)
    if cookie == "" || cookie != state {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid oauth state"})
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    tok, err := oauthCfg.Exchange(ctx, code)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "failed to exchange code"})
    }

    ghUser, primaryEmail, err := fetchGitHubUser(tok)
    if err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
    }

    // Link or create user
    user, err := linkOrCreateUserFromGitHub(ghUser, primaryEmail)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
    }

    // Issue JWT
    token, err := utils.GenerateJWT(user.ID)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate token"})
    }

    return c.JSON(AuthResponse{Token: token, User: *user})
}

// fetchGitHubUser retrieves the GitHub user profile and primary verified email.
func fetchGitHubUser(tok *oauth2.Token) (map[string]any, string, error) {
    client := &http.Client{Timeout: 10 * time.Second}
    // Fetch /user
    req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
    req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
    req.Header.Set("Accept", "application/vnd.github+json")
    resp, err := client.Do(req)
    if err != nil {
        return nil, "", err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        body, _ := io.ReadAll(resp.Body)
        return nil, "", fmt.Errorf("github /user failed: %s", string(body))
    }
    var profile map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
        return nil, "", err
    }

    // Determine email: prefer profile.email; else /user/emails
    email := ""
    if v, ok := profile["email"].(string); ok && v != "" {
        email = v
    }
    if email == "" {
        req2, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
        req2.Header.Set("Authorization", "Bearer "+tok.AccessToken)
        req2.Header.Set("Accept", "application/vnd.github+json")
        resp2, err := client.Do(req2)
        if err != nil {
            return nil, "", err
        }
        defer resp2.Body.Close()
        if resp2.StatusCode >= 300 {
            body, _ := io.ReadAll(resp2.Body)
            return nil, "", fmt.Errorf("github /user/emails failed: %s", string(body))
        }
        var emails []struct {
            Email    string `json:"email"`
            Primary  bool   `json:"primary"`
            Verified bool   `json:"verified"`
            Vis      string `json:"visibility"`
        }
        if err := json.NewDecoder(resp2.Body).Decode(&emails); err != nil {
            return nil, "", err
        }
        // choose primary verified, else first verified, else first
        for _, e := range emails {
            if e.Primary && e.Verified {
                email = e.Email
                break
            }
        }
        if email == "" {
            for _, e := range emails {
                if e.Verified {
                    email = e.Email
                    break
                }
            }
        }
        if email == "" && len(emails) > 0 {
            email = emails[0].Email
        }
    }

    if email == "" {
        return nil, "", errors.New("no email available from GitHub; ensure 'user:email' scope and a verified email")
    }

    return profile, email, nil
}

// linkOrCreateUserFromGitHub links an existing user or creates a new one from GitHub data.
func linkOrCreateUserFromGitHub(ghUser map[string]any, email string) (*models.User, error) {
    var user models.User

    // Extract fields
    var ghID string
    switch v := ghUser["id"].(type) {
    case float64:
        ghID = fmt.Sprintf("%0.f", v)
    case string:
        ghID = v
    default:
        ghID = fmt.Sprint(v)
    }
    login, _ := ghUser["login"].(string)
    avatar, _ := ghUser["avatar_url"].(string)

    // If user exists with this GitHubID, return it
    if err := config.DB.Where("git_hub_id = ?", ghID).First(&user).Error; err == nil {
        // Update avatar/provider if changed
        updates := map[string]any{"avatar_url": avatar, "provider": "github"}
        _ = config.DB.Model(&user).Updates(updates).Error
        return &user, nil
    }

    // Else try to find by email
    if err := config.DB.Where("email = ?", email).First(&user).Error; err == nil {
        updates := map[string]any{"git_hub_id": ghID, "avatar_url": avatar, "provider": "github"}
        if err := config.DB.Model(&user).Updates(updates).Error; err != nil {
            return nil, err
        }
        return &user, nil
    }

    // Create new user
    // Ensure unique username
    baseUsername := login
    if baseUsername == "" {
        baseUsername = strings.Split(email, "@")[0]
    }
    username := baseUsername
    for i := 0; i < 10; i++ {
        var count int64
        config.DB.Model(&models.User{}).Where("username = ?", username).Count(&count)
        if count == 0 {
            break
        }
        username = fmt.Sprintf("%s%d", baseUsername, i+1)
    }

    // Set a random hashed password to satisfy NOT NULL constraint
    rnd := make([]byte, 24)
    if _, err := rand.Read(rnd); err != nil {
        return nil, err
    }
    hashed, err := bcrypt.GenerateFromPassword([]byte(base64.RawURLEncoding.EncodeToString(rnd)), bcrypt.DefaultCost)
    if err != nil {
        return nil, err
    }

    newUser := models.User{
        Username:  username,
        Email:     email,
        Password:  string(hashed),
        Role:      "user",
        Provider:  "github",
        GitHubID:  ghID,
        AvatarURL: avatar,
    }
    if err := config.DB.Create(&newUser).Error; err != nil {
        return nil, err
    }
    return &newUser, nil
}

// Optional: debug endpoint to verify GitHub env
func GitHubConfigStatus(c *fiber.Ctx) error {
    if os.Getenv("GITHUB_CLIENT_ID") == "" || os.Getenv("GITHUB_CLIENT_SECRET") == "" || os.Getenv("GITHUB_REDIRECT_URL") == "" {
        return c.JSON(fiber.Map{"configured": false})
    }
    return c.JSON(fiber.Map{"configured": true})
}

