package organization

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Axontik/comin-leave-management-service/pkg/auth"
	"github.com/gin-gonic/gin"
)

type OrganizationClient struct {
	baseURL    string
	httpClient *http.Client
}

type OrganizationResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func NewOrganizationClient(baseURL string) *OrganizationClient {
	return &OrganizationClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (c *OrganizationClient) GetOrganization(token string, orgID string) (*OrganizationResponse, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/organizations/%s", c.baseURL, orgID), nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get organization")
	}

	var org OrganizationResponse
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, err
	}

	return &org, nil
}

// Middleware to validate requests
func ValidateOrganizationAccess(authClient *auth.AuthClient, orgClient *OrganizationClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		user, err := authClient.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// Check if organization exists and is active
		org, err := orgClient.GetOrganization(string(token), string(user.OrganizationID))
		log.Printf("Organization Client: %+v", org)
		if err != nil || org.Status != "active" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid organization access"})
			return
		}

		c.Set("user_id", user.UserID)
		c.Set("organization_id", user.OrganizationID)
		c.Set("email", user.Email)
		c.Set("role", user.Role)

		c.Next()
	}
}
