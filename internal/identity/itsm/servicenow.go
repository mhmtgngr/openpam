package itsm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

const (
	serviceNowTable = "incident"
	serviceNowAPIBase = "/api/now/table"
)

// ServiceNowClient integrates with ServiceNow for access request workflows
type ServiceNowClient struct {
	instanceURL string
	username    string
	password    string
	client      *http.Client
	logger      *zerolog.Logger
}

// ServiceNowConfig holds ServiceNow connection configuration
type ServiceNowConfig struct {
	InstanceURL string
	Username    string
	Password    string
	Timeout     time.Duration
}

// NewServiceNowClient creates a new ServiceNow client
func NewServiceNowClient(instanceURL, username, password string, logger *zerolog.Logger) *ServiceNowClient {
	return &ServiceNowClient{
		instanceURL: instanceURL,
		username:    username,
		password:    password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ServiceNowIncident represents a ServiceNow incident
type ServiceNowIncident struct {
	SysID           string                 `json:"sys_id"`
	Number          string                 `json:"number"`
	ShortDescription string                `json:"short_description"`
	Description     string                 `json:"description"`
	Priority        int                    `json:"priority"`
	State           int                    `json:"state"`
	Urgency         int                    `json:"urgency"`
	Impact          int                    `json:"impact"`
	AssignmentGroup string                 `json:"assignment_group"`
	AssignedTo      string                 `json:"assigned_to"`
	CallerID        string                 `json:"caller_id"`
	WorkNotes       string                 `json:"work_notes"`
	Comments        string                 `json:"comments"`
	CorrelationID   string                 `json:"correlation_id,omitempty"`
	// Custom fields
	UEventType      string                 `json:"u_event_type,omitempty"`
	URequestID      string                 `json:"u_request_id,omitempty"`
	UAccessType     string                 `json:"u_access_type,omitempty"`
	UDuration       string                 `json:"u_duration,omitempty"`
}

// ServiceNowResponse represents a ServiceNow API response
type ServiceNowResponse struct {
	Result []ServiceNowIncident `json:"result"`
	Status string               `json:"status"`
	Error  *ServiceNowError     `json:"error"`
}

// ServiceNowError represents a ServiceNow error
type ServiceNowError struct {
	Detail    string `json:"detail"`
	Message   string `json:"message"`
	ErrorCode string `json:"error_code"`
}

// ServiceNowStatus maps ServiceNow states to our status
var ServiceNowStatusMap = map[int]string{
	1: "new",           // New
	2: "in_progress",   // In Progress
	3: "in_progress",   // Pending
	4: "pending",       // Await Customer
	5: "pending",       // Await Vendor
	6: "approved",      // Resolved
	7: "closed",        // Closed
	8: "cancelled",     // Canceled
}

// CreateTicket creates a new incident in ServiceNow
func (c *ServiceNowClient) CreateTicket(ctx context.Context, req *ITSMTicketCreate) (*ITSMTicket, error) {
	incident := ServiceNowIncident{
		ShortDescription: req.Title,
		Description:      req.Description,
		State:           1, // New
		UEventType:      "access_request",
	}

	// Set priority based on request priority
	switch req.Priority {
	case "critical":
		incident.Priority = 1
		incident.Urgency = 1
		incident.Impact = 1
	case "high":
		incident.Priority = 2
		incident.Urgency = 1
		incident.Impact = 2
	case "medium":
		incident.Priority = 3
		incident.Urgency = 2
		incident.Impact = 2
	default:
		incident.Priority = 4
		incident.Urgency = 3
		incident.Impact = 3
	}

	// Add metadata
	if requestID, ok := req.Metadata["request_id"]; ok {
		incident.URequestID = requestID
		incident.CorrelationID = requestID
	}
	if accessType, ok := req.Metadata["access_type"]; ok {
		incident.UAccessType = accessType
	}
	if duration, ok := req.Metadata["duration"]; ok {
		incident.UDuration = duration
	}

	url := fmt.Sprintf("%s%s/%s", c.instanceURL, serviceNowAPIBase, serviceNowTable)

	body, err := json.Marshal(map[string]ServiceNowIncident{"result": incident})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.SetBasicAuth(c.username, c.password)
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
			Msg("ServiceNow API error")
		return nil, fmt.Errorf("ServiceNow API error: %s", resp.Status)
	}

	var snResp ServiceNowResponse
	if err := json.Unmarshal(respBody, &snResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(snResp.Result) == 0 {
		return nil, fmt.Errorf("no incident returned")
	}

	incidentResp := snResp.Result[0]

	ticket := &ITSMTicket{
		ID:     incidentResp.Number,
		Status: "new",
		URL:    fmt.Sprintf("%s/now/nav/ui/classic/params/target/%s", c.instanceURL, incidentResp.SysID),
		Metadata: map[string]string{
			"sys_id":       incidentResp.SysID,
			"sys_class_name": "incident",
			"type":         "servicenow",
		},
	}

	c.logger.Info().
		Str("ticket_number", ticket.ID).
		Str("sys_id", incidentResp.SysID).
		Msg("ServiceNow incident created")

	return ticket, nil
}

// GetTicketStatus retrieves the status of a ServiceNow incident
func (c *ServiceNowClient) GetTicketStatus(ctx context.Context, ticketID string) (string, error) {
	url := fmt.Sprintf("%s%s/%s?number=%s", c.instanceURL, serviceNowAPIBase, serviceNowTable, ticketID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.SetBasicAuth(c.username, c.password)
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
		return "", fmt.Errorf("ServiceNow API error: %s", resp.Status)
	}

	var snResp ServiceNowResponse
	if err := json.Unmarshal(respBody, &snResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(snResp.Result) == 0 {
		return "", fmt.Errorf("ticket not found")
	}

	incident := snResp.Result[0]

	// Map ServiceNow state to our status
	status, ok := ServiceNowStatusMap[incident.State]
	if !ok {
		status = "unknown"
	}

	return status, nil
}

// UpdateTicket updates a ServiceNow incident
func (c *ServiceNowClient) UpdateTicket(ctx context.Context, ticketID string, status string, notes string) error {
	// Map our status to ServiceNow state
	state := 2 // Default to In Progress
	switch status {
	case "approved":
		state = 6 // Resolved
	case "active":
		state = 2 // In Progress
	case "denied":
		state = 8 // Canceled
	case "cancelled":
		state = 8 // Canceled
	case "revoked":
		state = 8 // Canceled
	}

	url := fmt.Sprintf("%s%s/%s?number=%s", c.instanceURL, serviceNowAPIBase, serviceNowTable, ticketID)

	update := ServiceNowIncident{
		State:     state,
		WorkNotes: notes,
	}

	body, err := json.Marshal(map[string]ServiceNowIncident{"result": update})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.SetBasicAuth(c.username, c.password)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		c.logger.Error().
			Str("status", resp.Status).
			Str("response", string(respBody)).
			Msg("ServiceNow update failed")
		return fmt.Errorf("ServiceNow API error: %s", resp.Status)
	}

	c.logger.Info().
		Str("ticket_number", ticketID).
		Str("status", status).
		Msg("ServiceNow incident updated")

	return nil
}

// AddComment adds a comment to a ServiceNow incident
func (c *ServiceNowClient) AddComment(ctx context.Context, ticketID string, comment string) error {
	url := fmt.Sprintf("%s%s/%s?number=%s", c.instanceURL, serviceNowAPIBase, serviceNowTable, ticketID)

	update := ServiceNowIncident{
		Comments: comment,
	}

	body, err := json.Marshal(map[string]ServiceNowIncident{"result": update})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.SetBasicAuth(c.username, c.password)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ServiceNow API error: %s", resp.Status)
	}

	return nil
}

// Type returns the ITSM client type
func (c *ServiceNowClient) Type() string {
	return "servicenow"
}

// ITSMTicketCreate and ITSMTicket are defined in access_request.go but need to be accessible
// These types should be in a shared package, but for now we'll define them here

type ITSMTicketCreate struct {
	Title       string
	Description string
	Requester   string
	Priority    string
	Type        string
	Metadata    map[string]string
}

type ITSMTicket struct {
	ID       string
	Status   string
	URL      string
	Metadata map[string]string
}
