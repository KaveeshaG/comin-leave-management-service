// pkg/auth/client.go
package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type AuthClient struct {
	baseURL    string
	httpClient *http.Client
}

type UserResponse struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewAuthClient(baseURL string) *AuthClient {
	return &AuthClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (c *AuthClient) ValidateToken(token string) (*UserResponse, error) {
	log.Printf("Validating token: %s", token)

	token = strings.TrimPrefix(token, "Bearer ")

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/validate", c.baseURL), nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	log.Printf("Making request to: %s with Authorization: %s", req.URL.String(), req.Header.Get("Authorization"))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Printf("Response status: %d, body: %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("auth service error: status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("auth service error: %s", errResp.Error)
	}

	var claims UserResponse
	if err := json.Unmarshal(body, &claims); err != nil {
		return nil, err
	}

	return &claims, nil
}

func (c *AuthClient) RefreshToken(refreshToken string) (*TokenResponse, error) {
	payload := map[string]string{
		"refresh_token": refreshToken,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/auth/refresh", c.baseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("auth service error: %s", errResp.Error)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// Helper method to verify organization access
func (c *AuthClient) VerifyOrganizationAccess(token string, organizationID string) error {
	claims, err := c.ValidateToken(token)
	if err != nil {
		return err
	}

	if claims.OrganizationID != organizationID {
		return fmt.Errorf("user does not have access to this organization")
	}

	return nil
}

// Helper method to verify admin role
func (c *AuthClient) VerifyAdminRole(token string) error {
	claims, err := c.ValidateToken(token)
	if err != nil {
		return err
	}

	if claims.Role != "admin" {
		return fmt.Errorf("user does not have admin privileges")
	}

	return nil
}

// Add middleware for authentication
type AuthMiddleware struct {
	authClient *AuthClient
}

func NewAuthMiddleware(authClient *AuthClient) *AuthMiddleware {
	return &AuthMiddleware{
		authClient: authClient,
	}
}

func (m *AuthMiddleware) RequireAuth(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "missing authorization header"})
		return
	}

	// Remove "Bearer " prefix if present
	token = strings.TrimPrefix(token, "Bearer ")

	claims, err := m.authClient.ValidateToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	// Set claims in context for later use
	c.Set("user_id", claims.UserID)
	c.Set("org_id", claims.OrganizationID)
	c.Set("email", claims.Email)
	c.Set("role", claims.Role)

	c.Next()
}

func (m *AuthMiddleware) RequireOrganizationAccess(c *gin.Context) {
	token := c.GetHeader("Authorization")
	orgID := c.Param("org_id") // Make sure this matches your route parameter

	log.Printf("Checking access - Token: %s, OrgID: %s", token, orgID)

	if err := m.authClient.VerifyOrganizationAccess(token, orgID); err != nil {
		log.Printf("Access verification failed: %v", err)
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "no access to this organization"})
		return
	}

	c.Next()
}

func (m *AuthMiddleware) RequireAdmin(c *gin.Context) {
	token := c.GetHeader("Authorization")

	if err := m.authClient.VerifyAdminRole(token); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "admin privileges required"})
		return
	}

	c.Next()
}
