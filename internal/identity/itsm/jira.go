package itsm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

const (
	jiraAPIBase = "/rest/api/3"
	jiraIssueType = "Service Request"
)

// JiraClient integrates with Atlassian Jira for access request workflows
type JiraClient struct {
	baseURL   string
	username  string
	apiToken  string
	projectKey string
	client    *http.Client
	logger    *zerolog.Logger
}

// JiraConfig holds Jira connection configuration
type JiraConfig struct {
	BaseURL    string
	Username   string
	APIToken   string
	ProjectKey string
	Timeout    time.Duration
}

// NewJiraClient creates a new Jira client
func NewJiraClient(baseURL, username, apiToken, projectKey string, logger *zerolog.Logger) *JiraClient {
	return &JiraClient{
		baseURL:    baseURL,
		username:   username,
		apiToken:   apiToken,
		projectKey: projectKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// JiraIssue represents a Jira issue
type JiraIssue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Self   string `json:"self"`
	Fields JiraFields `json:"fields"`
}

// JiraFields represents the fields of a Jira issue
type JiraFields struct {
	Summary     string              `json:"summary"`
	Description *JiraDocument       `json:"description"`
	IssueType   JiraIssueType       `json:"issuetype"`
	Project     JiraProject         `json:"project"`
	Priority    *JiraPriority       `json:"priority"`
	Status      JiraStatus          `json:"status"`
	Reporter    *JiraUser           `json:"reporter"`
	Assignee    *JiraUser           `json:"assignee"`
	Labels      []string            `json:"labels"`
	CustomFields map[string]interface{} `json:"-"`
}

// JiraDocument represents a Jira document (Atlassian Document Format)
type JiraDocument struct {
	Version int           `json:"version"`
	Type    string        `json:"type"`
	Content []JiraContent `json:"content"`
}

// JiraContent represents content in a Jira document
type JiraContent struct {
	Type    string              `json:"type"`
	Text    string              `json:"text,omitempty"`
	Content []JiraContent       `json:"content,omitempty"`
	Attrs   map[string]string   `json:"attrs,omitempty"`
}

// JiraIssueType represents a Jira issue type
type JiraIssueType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Self    string `json:"self"`
}

// JiraProject represents a Jira project
type JiraProject struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Name   string `json:"name"`
	Self   string `json:"self"`
}

// JiraPriority represents a Jira priority
type JiraPriority struct {
	Self    string `json:"self"`
	IconURL string `json:"iconUrl"`
	Name    string `json:"name"`
	ID      string `json:"id"`
}

// JiraStatus represents a Jira status
type JiraStatus struct {
	Self           string         `json:"self"`
	Description    string         `json:"description"`
	IconURL        string         `json:"iconUrl"`
	Name           string         `json:"name"`
	ID             string         `json:"id"`
	StatusCategory JiraStatusCat  `json:"statusCategory"`
}

// JiraStatusCat represents a Jira status category
type JiraStatusCat struct {
	Self      string `json:"self"`
	ID        int    `json:"id"`
	Key       string `json:"key"`
	ColorName string `json:"colorName"`
	Name      string `json:"name"`
}

// JiraUser represents a Jira user
type JiraUser struct {
	Self         string `json:"self"`
	AccountID    string `json:"accountId"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
	Active       bool   `json:"active"`
}

// JiraCreateRequest represents a request to create a Jira issue
type JiraCreateRequest struct {
	Fields JiraFields `json:"fields"`
}

// JiraCreateResponse represents the response from creating a Jira issue
type JiraCreateResponse struct {
	ID    string     `json:"id"`
	Key   string     `json:"key"`
	Self  string     `json:"self"`
}

// JiraStatusMap maps Jira status categories to our status
var JiraStatusMap = map[string]string{
	"new":        "new",
	"indeterminate": "in_progress",
	"done":       "approved",
}

// CreateTicket creates a new issue in Jira
func (c *JiraClient) CreateTicket(ctx context.Context, req *ITSMTicketCreate) (*ITSMTicket, error) {
	// Build description in Atlassian Document Format
	description := &JiraDocument{
		Version: 1,
		Type:    "doc",
		Content: []JiraContent{
			{
				Type: "paragraph",
				Content: []JiraContent{
					{Type: "text", Text: req.Description},
				},
			},
		},
	}

	// Map priority
	priorityID := "3" // Medium
	switch req.Priority {
	case "critical":
		priorityID = "1" // Highest
	case "high":
		priorityID = "2" // High
	case "low":
		priorityID = "4" // Low
	}

	// Add labels for metadata
	labels := []string{"access-request", "openpam"}
	if reqType, ok := req.Metadata["access_type"]; ok {
		labels = append(labels, reqType)
	}

	createReq := JiraCreateRequest{
		Fields: JiraFields{
			Project: JiraProject{
				Key: c.projectKey,
			},
			Summary: req.Title,
			Description: description,
			IssueType: JiraIssueType{
				Name: "Service Request",
			},
			Priority: &JiraPriority{
				ID: priorityID,
			},
			Labels: labels,
		},
	}

	body, err := json.Marshal(createReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s/issue", c.baseURL, jiraAPIBase)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", c.basicAuth())
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		c.logger.Error().
			Str("status", resp.Status).
			Str("response", string(respBody)).
			Msg("Jira API error")
		return nil, fmt.Errorf("Jira API error: %s", resp.Status)
	}

	var createResp JiraCreateResponse
	if err := json.Unmarshal(respBody, &createResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	ticket := &ITSMTicket{
		ID:     createResp.Key,
		Status: "new",
		URL:    fmt.Sprintf("%s/browse/%s", c.baseURL, createResp.Key),
		Metadata: map[string]string{
			"issue_id": createResp.ID,
			"type":     "jira",
		},
	}

	c.logger.Info().
		Str("ticket_key", ticket.ID).
		Str("issue_id", createResp.ID).
		Msg("Jira issue created")

	return ticket, nil
}

// GetTicketStatus retrieves the status of a Jira issue
func (c *JiraClient) GetTicketStatus(ctx context.Context, ticketID string) (string, error) {
	url := fmt.Sprintf("%s%s/issue/%s", c.baseURL, jiraAPIBase, ticketID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", c.basicAuth())
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Jira API error: %s", resp.Status)
	}

	var issue JiraIssue
	if err := json.Unmarshal(respBody, &issue); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	// Map Jira status category to our status
	statusCatKey := issue.Fields.Status.StatusCategory.Key
	status, ok := JiraStatusMap[statusCatKey]
	if !ok {
		status = "unknown"
	}

	// Check specific status names for more granular mapping
	statusName := issue.Fields.Status.Name
	switch statusName {
	case "Cancelled", "Canceled", "Closed":
		status = "cancelled"
	case "Approved", "Done":
		status = "approved"
	case "Rejected", "Declined":
		status = "denied"
	}

	return status, nil
}

// UpdateTicket updates a Jira issue
func (c *JiraClient) UpdateTicket(ctx context.Context, ticketID string, status string, notes string) error {
	// Map our status to Jira transition
	// This would typically require getting the available transitions first
	// For simplicity, we'll add a comment instead

	if notes != "" {
		return c.AddComment(ctx, ticketID, notes)
	}

	return nil
}

// AddComment adds a comment to a Jira issue
func (c *JiraClient) AddComment(ctx context.Context, ticketID string, comment string) error {
	commentDoc := map[string]interface{}{
		"body": JiraDocument{
			Version: 1,
			Type:    "doc",
			Content: []JiraContent{
				{
					Type: "paragraph",
					Content: []JiraContent{
						{Type: "text", Text: comment},
					},
				},
			},
		},
	}

	body, err := json.Marshal(commentDoc)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s%s/issue/%s/comment", c.baseURL, jiraAPIBase, ticketID)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", c.basicAuth())
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Jira API error: %s", resp.Status)
	}

	return nil
}

// Type returns the ITSM client type
func (c *JiraClient) Type() string {
	return "jira"
}

// basicAuth generates the basic auth header value
func (c *JiraClient) basicAuth() string {
	auth := fmt.Sprintf("%s:%s", c.username, c.apiToken)
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(auth)))
}

// TransitionIssue transitions an issue to a new status
func (c *JiraClient) TransitionIssue(ctx context.Context, ticketID string, transitionName string) error {
	// First, get available transitions
	url := fmt.Sprintf("%s%s/issue/%s/transitions", c.baseURL, jiraAPIBase, ticketID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", c.basicAuth())
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var transitionsResp struct {
		Transitions []struct {
			ID   interface{} `json:"id"`
			Name string      `json:"name"`
		} `json:"transitions"`
	}

	if err := json.Unmarshal(respBody, &transitionsResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	// Find the transition ID
	var transitionID interface{}
	for _, t := range transitionsResp.Transitions {
		if t.Name == transitionName {
			transitionID = t.ID
			break
		}
	}

	if transitionID == nil {
		return fmt.Errorf("transition '%s' not found", transitionName)
	}

	// Execute the transition
	transitionReq := map[string]interface{}{
		"transition": map[string]interface{}{
			"id": transitionID,
		},
	}

	body, err := json.Marshal(transitionReq)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", c.basicAuth())
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err = c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ = io.ReadAll(resp.Body)
		return fmt.Errorf("Jira API error: %s - %s", resp.Status, string(respBody))
	}

	return nil
}
