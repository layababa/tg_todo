package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	notionAuthURL  = "https://api.notion.com/v1/oauth/authorize"
	notionTokenURL = "https://api.notion.com/v1/oauth/token"
)

// OAuthConfig holds the Notion OAuth configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// TokenResponse represents the response from Notion's token exchange
type TokenResponse struct {
	AccessToken          string          `json:"access_token"`
	TokenType            string          `json:"token_type"`
	BotID                string          `json:"bot_id"`
	WorkspaceID          string          `json:"workspace_id"`
	WorkspaceName        string          `json:"workspace_name"`
	WorkspaceIcon        string          `json:"workspace_icon"`
	Owner                json.RawMessage `json:"owner"`
	DuplicatedTemplateID string          `json:"duplicated_template_id,omitempty"`
}

// GenerateAuthURL generates the Notion OAuth authorization URL
// state: A random string to prevent CSRF attacks
func GenerateAuthURL(config OAuthConfig, state string) string {
	params := url.Values{}
	params.Add("client_id", config.ClientID)
	params.Add("redirect_uri", config.RedirectURI)
	params.Add("response_type", "code")
	params.Add("owner", "user")
	params.Add("state", state)

	return notionAuthURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for an access token
func ExchangeCode(ctx context.Context, config OAuthConfig, code string) (*TokenResponse, error) {
	// Prepare request body
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", config.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", notionTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(config.ClientID, config.ClientSecret)

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tokenResp, nil
}
