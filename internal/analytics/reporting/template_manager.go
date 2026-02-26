package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/analytics"
	"github.com/rs/zerolog"
)

// TemplateManager manages pre-built compliance report templates
type TemplateManager struct {
	db     *sqlx.DB
	logger zerolog.Logger
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(db *sqlx.DB, logger zerolog.Logger) *TemplateManager {
	return &TemplateManager{
		db:     db,
		logger: logger,
	}
}

// ReportTemplate represents a compliance report template
type ReportTemplate struct {
	ID          uuid.UUID              `db:"id" json:"id"`
	TenantID    *uuid.UUID             `db:"tenant_id" json:"tenant_id"`
	Name        string                 `db:"name" json:"name"`
	Framework   analytics.ComplianceFramework `db:"framework" json:"framework"`
	Version     string                 `db:"version" json:"version"`
	Description string                 `db:"description" json:"description"`
	// JSONB fields - stored as bytes in DB, unmarshaled for use
	templateDataBytes   []byte            `db:"template_data"`
	sectionsBytes       []byte            `db:"sections"`
	TemplateData        map[string]interface{} `json:"template_data"`
	Sections            []TemplateSection       `json:"sections"`
	IsPublic            bool                   `db:"is_public" json:"is_public"`
	CreatedBy           uuid.UUID              `db:"created_by" json:"created_by"`
	CreatedAt           time.Time              `db:"created_at" json:"created_at"`
}

// TemplateSection represents a section in a report template
type TemplateSection struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"` // summary, controls, findings, recommendations
	Order       int                    `json:"order"`
	Content     map[string]interface{} `json:"content"`
}

// GetTemplate retrieves a report template
func (m *TemplateManager) GetTemplate(ctx context.Context, id uuid.UUID) (*ReportTemplate, error) {
	query := `
		SELECT * FROM report_templates
	WHERE id = $1 AND deleted_at IS NULL
	`

	var template ReportTemplate
	err := m.db.GetContext(ctx, &template, query, id)
	if err != nil {
		return nil, fmt.Errorf("template_manager.GetTemplate: %w", err)
	}

	// Unmarshal JSON fields from byte arrays
	if len(template.templateDataBytes) > 0 {
		if err := json.Unmarshal(template.templateDataBytes, &template.TemplateData); err != nil {
			return nil, err
		}
	}
	if len(template.sectionsBytes) > 0 {
		if err := json.Unmarshal(template.sectionsBytes, &template.Sections); err != nil {
			return nil, err
		}
	}

	return &template, nil
}

// ListTemplates lists report templates for a tenant
func (m *TemplateManager) ListTemplates(ctx context.Context, tenantID *uuid.UUID, framework *analytics.ComplianceFramework) ([]ReportTemplate, error) {
	query := `
		SELECT * FROM report_templates
		WHERE deleted_at IS NULL
	`
	args := []interface{}{}

	if tenantID != nil {
		query += " AND (tenant_id = $1 OR tenant_id IS NULL)"
		args = append(args, *tenantID)
	} else {
		query += " AND tenant_id IS NULL" // Only show global templates
	}

	if framework != nil {
		query += " AND framework = $2"
		args = append(args, string(*framework))
	}

	query += " ORDER BY framework, name"

	var templates []ReportTemplate
	err := m.db.SelectContext(ctx, &templates, query, args...)
	if err != nil {
		return nil, fmt.Errorf("template_manager.ListTemplates: %w", err)
	}

	// Unmarshal JSON fields from byte arrays
	for i := range templates {
		if len(templates[i].templateDataBytes) > 0 {
			json.Unmarshal(templates[i].templateDataBytes, &templates[i].TemplateData)
		}
		if len(templates[i].sectionsBytes) > 0 {
			json.Unmarshal(templates[i].sectionsBytes, &templates[i].Sections)
		}
	}

	return templates, nil
}

// CreateTemplate creates a new report template
func (m *TemplateManager) CreateTemplate(ctx context.Context, template *ReportTemplate) error {
	template.ID = uuid.New()
	template.CreatedAt = time.Now()

	templateData, _ := json.Marshal(template.TemplateData)
	sections, _ := json.Marshal(template.Sections)

	query := `
		INSERT INTO report_templates (
			id, tenant_id, name, framework, version, description,
			template_data, sections, is_public, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := m.db.ExecContext(ctx, query,
		template.ID, template.TenantID, template.Name, string(template.Framework),
		template.Version, template.Description, templateData, sections,
		template.IsPublic, template.CreatedBy, template.CreatedAt,
	)

	return err
}

// GetFrameworkTemplate gets the default template for a framework
func (m *TemplateManager) GetFrameworkTemplate(ctx context.Context, framework analytics.ComplianceFramework) (*ReportTemplate, error) {
	templates, err := m.ListTemplates(ctx, nil, &framework)
	if err != nil {
		return nil, err
	}

	if len(templates) == 0 {
		// Create default template
		return m.createDefaultFrameworkTemplate(ctx, framework)
	}

	return &templates[0], nil
}

// createDefaultFrameworkTemplate creates a default template for a framework
func (m *TemplateManager) createDefaultFrameworkTemplate(ctx context.Context, framework analytics.ComplianceFramework) (*ReportTemplate, error) {
	template := &ReportTemplate{
		TenantID:    nil, // Global template
		Name:        fmt.Sprintf("Default %s Template", framework),
		Framework:   framework,
		Version:     "1.0",
		Description: fmt.Sprintf("Default %s compliance report template", framework),
		TemplateData: make(map[string]interface{}),
		Sections:     m.getDefaultFrameworkSections(framework),
		IsPublic:    true,
		CreatedBy:   uuid.Nil, // System created
		CreatedAt:    time.Now(),
	}

	// Add framework-specific controls
	template.TemplateData["controls"] = m.getFrameworkControls(framework)

	return template, nil
}

// getDefaultFrameworkSections returns default sections for a framework
func (m *TemplateManager) getDefaultFrameworkSections(framework analytics.ComplianceFramework) []TemplateSection {
	baseSections := []TemplateSection{
		{
			ID:          "summary",
			Title:       "Executive Summary",
			Description: "High-level overview of compliance status",
			Type:        "summary",
			Order:       1,
			Content:     map[string]interface{}{},
		},
		{
			ID:          "scope",
			Title:       "Scope of Assessment",
			Description: "Systems and processes covered in this assessment",
			Type:        "findings",
			Order:       2,
			Content:     map[string]interface{}{},
		},
		{
			ID:          "controls",
			Title:       "Control Evaluations",
			Description: "Detailed results for each evaluated control",
			Type:        "controls",
			Order:       3,
			Content:     map[string]interface{}{},
		},
		{
			ID:          "findings",
			Title:       "Findings and Recommendations",
			Description: "Issues identified and remediation steps",
			Type:        "findings",
			Order:       4,
			Content:     map[string]interface{}{},
		},
		{
			ID:          "evidence",
			Title:       "Evidence Repository",
			Description: "Supporting evidence for control evaluations",
			Type:        "evidence",
			Order:       5,
			Content:     map[string]interface{}{},
		},
	}

	// Add framework-specific sections
	switch framework {
	case analytics.FrameworkSOC2:
		baseSections = append(baseSections, TemplateSection{
			ID:          "trust_criteria",
			Title:       "Trust Criteria",
			Description: "SOC 2 Trust Criteria evaluation",
			Type:        "controls",
			Order:       6,
			Content:     map[string]interface{}{},
		})
	case analytics.FrameworkISO27001:
		baseSections = append(baseSections, TemplateSection{
			ID:          "annex_a",
			Title:       "Statement of Applicability",
			Description: "ISO 27001 Statement of Applicability",
			Type:        "findings",
			Order:       6,
			Content:     map[string]interface{}{},
		})
	case analytics.FrameworkPCIDSS:
		baseSections = append(baseSections, TemplateSection{
			ID:          "saq",
			Title:       "SAQ Requirements",
			Description: "PCI DSS SAQ compliance details",
			Type:        "controls",
			Order:       6,
			Content:     map[string]interface{}{},
		})
	}

	return baseSections
}

// getFrameworkControls returns default controls for a framework
func (m *TemplateManager) getFrameworkControls(framework analytics.ComplianceFramework) []ControlMapping {
	controls := make([]ControlMapping, 0)

	switch framework {
	case analytics.FrameworkSOC2:
		controls = append(controls, getSOC2Controls()...)
	case analytics.FrameworkISO27001:
		controls = append(controls, getISO27001Controls()...)
	case analytics.FrameworkPCIDSS:
		controls = append(controls, getPCIDSSControls()...)
	case analytics.FrameworkHIPAA:
		controls = append(controls, getHIPAAControls()...)
	case analytics.FrameworkNIST:
		controls = append(controls, getNISTControls()...)
	case analytics.FrameworkGDPR:
		controls = append(controls, getGDPRControls()...)
	}

	return controls
}

// ControlMapping maps a requirement to OpenPAM capabilities
type ControlMapping struct {
	ControlID      string
	ControlName    string
	Category       string
	OpenPAMFeature string
	EvidenceSources []string
	TestProcedure string
}

// getSOC2Controls returns SOC 2 controls
func getSOC2Controls() []ControlMapping {
	return []ControlMapping{
		{
			ControlID:      "AC-1.1",
			ControlName:    "Access Control Policy",
			Category:       "Access Control",
			OpenPAMFeature: "access_requests",
			EvidenceSources: []string{"access_requests", "approvals"},
			TestProcedure: "Verify all access is approved through formal requests",
		},
		{
			ControlID:      "AC-2.1",
			ControlName:    "Asset Inventory",
			Category:       "Asset Management",
			OpenPAMFeature: "discovery",
			EvidenceSources: []string{"discovery_targets"},
			TestProcedure: "Verify all assets are discovered and catalogued",
		},
		{
			ControlID:      "AU-2.1",
			ControlName: "Audit Logging",
			Category:       "Audit",
			OpenPAMFeature: "audit_logs",
			EvidenceSources: []string{"audit_logs", "session_analytics"},
			TestProcedure: "Verify all privileged actions are logged",
		},
		{
			ControlID:      "AU-3.1",
			ControlName: "Monitoring",
			Category:       "Monitoring",
			OpenPAMFeature: "anomaly_detection",
			EvidenceSources: []string{"anomaly_detections", "alerts"},
			TestProcedure: "Verify suspicious activity is detected and alerted",
		},
	}
}

// getISO27001Controls returns ISO 27001 controls
func getISO27001Controls() []ControlMapping {
	return []ControlMapping{
		{
			ControlID:      "A.9.1.1",
			ControlName:    "Access Control Policy",
			Category:       "Access Control",
			OpenPAMFeature: "access_requests",
			EvidenceSources: []string{"access_requests", "approvals"},
			TestProcedure: "Verify access is granted based on approval",
		},
		{
			ControlID:      "A.12.3.1",
			ControlName: "Backup",
			Category:       "Operations Security",
			OpenPAMFeature: "session_recording",
			EvidenceSources: []string{"session_recordings"},
			TestProcedure: "Verify sessions are recorded and stored securely",
		},
	}
}

// getPCIDSSControls returns PCI DSS controls
func getPCIDSSControls() []ControlMapping {
	return []ControlMapping{
		{
			ControlID:      "7.1",
			ControlName:    "Limit Access",
			Category:       "Access Control",
			OpenPAMFeature: "access_requests",
			EvidenceSources: []string{"access_requests"},
			TestProcedure: "Verify access is limited to need-to-know",
		},
		{
			ControlID:      "7.2",
			ControlName: "Unique Identification",
			Category:       "Access Control",
			OpenPAMFeature: "mfa",
			EvidenceSources: []string{"mfa_logs"},
			TestProcedure: "Verify MFA is required for all access",
		},
	}
}

// getHIPAAControls returns HIPAA controls
func getHIPAAControls() []ControlMapping {
	return []ControlMapping{
		{
			ControlID:      "164.312(a)(2)(i)",
			ControlName: "Access Control",
			Category:       "Access Control",
			OpenPAMFeature: "access_requests",
			EvidenceSources: []string{"access_requests", "approvals"},
			TestProcedure: "Verify access is controlled and logged",
		},
		{
			ControlID:      "164.312(b)",
			ControlName: "Audit Controls",
			Category:       "Audit",
			OpenPAMFeature: "audit_logs",
			EvidenceSources: []string{"audit_logs", "session_analytics"},
			TestProcedure: "Verify all access is audited",
		},
	}
}

// getNISTControls returns NIST 800-53 controls
func getNISTControls() []ControlMapping {
	return []ControlMapping{
		{
			ControlID:      "AC-6",
			ControlName: "Least Privilege",
			Category:       "Access Control",
			OpenPAMFeature: "jit_access",
			EvidenceSources: []string{"access_requests", "sessions"},
			TestProcedure: "Verify just-in-time access is enforced",
		},
	}
}

// getGDPRControls returns GDPR controls
func getGDPRControls() []ControlMapping {
	return []ControlMapping{
		{
			ControlID:      "Art.32",
			ControlName: "Security of Processing",
			Category:       "Security",
			OpenPAMFeature: "encryption",
			EvidenceSources: []string{"audit_logs"},
			TestProcedure: "Verify data is encrypted at rest and in transit",
		},
	}
}
