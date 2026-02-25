// Package reports provides HTML templates for report generation
package reports

import (
	"fmt"
	"html/template"
	"time"
)

// Template functions
var templateFuncs = template.FuncMap{
	"formatDate":   formatDate,
	"formatTime":   formatTime,
	"formatNumber": formatNumber,
	"formatPercent": formatPercent,
	"severityColor": severityColor,
	"scoreColor":    scoreColor,
}

// GetComplianceReportTemplate returns the compliance report HTML template
func GetComplianceReportTemplate() *template.Template {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Compliance Report - {{.Score.Framework}}</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: #f5f7fa;
            color: #2c3e50;
            line-height: 1.6;
            padding: 20px;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 30px;
        }
        .header h1 { font-size: 28px; margin-bottom: 10px; }
        .header .meta { opacity: 0.9; font-size: 14px; }
        .score-banner {
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 40px;
            background: {{scoreColor .Score.OverallScore}};
            color: white;
        }
        .score-circle {
            width: 200px;
            height: 200px;
            border-radius: 50%;
            border: 8px solid rgba(255,255,255,0.3);
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
        }
        .score-value { font-size: 64px; font-weight: bold; }
        .score-label { font-size: 14px; opacity: 0.9; }
        .content { padding: 30px; }
        .section { margin-bottom: 40px; }
        .section-title {
            font-size: 20px;
            font-weight: 600;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 2px solid #e1e8ed;
        }
        .control-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
        }
        .control-card {
            border: 1px solid #e1e8ed;
            border-radius: 8px;
            padding: 20px;
        }
        .control-card.passed { border-left: 4px solid #10b981; }
        .control-card.warning { border-left: 4px solid #f59e0b; }
        .control-card.failed { border-left: 4px solid #ef4444; }
        .control-name { font-weight: 600; margin-bottom: 10px; }
        .score-bar {
            height: 8px;
            background: #e1e8ed;
            border-radius: 4px;
            overflow: hidden;
            margin: 10px 0;
        }
        .score-fill {
            height: 100%;
            border-radius: 4px;
            transition: width 0.3s ease;
        }
        .findings-list { margin-top: 15px; }
        .finding {
            padding: 10px;
            margin: 5px 0;
            background: #f8fafc;
            border-radius: 4px;
            font-size: 13px;
        }
        .finding.critical { background: #fef2f2; border-left: 3px solid #dc2626; }
        .finding.high { background: #fff7ed; border-left: 3px solid #ea580c; }
        .finding.medium { background: #fefce8; border-left: 3px solid #ca8a04; }
        .finding.low { background: #f0fdf4; border-left: 3px solid #16a34a; }
        .footer {
            background: #f8fafc;
            padding: 20px 30px;
            text-align: center;
            font-size: 12px;
            color: #64748b;
        }
        .badge {
            display: inline-block;
            padding: 2px 8px;
            border-radius: 12px;
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
        }
        .badge.passed { background: #d1fae5; color: #065f46; }
        .badge.warning { background: #fef3c7; color: #92400e; }
        .badge.failed { background: #fee2e2; color: #991b1b; }
        .summary-stats {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            text-align: center;
            padding: 20px;
            background: #f8fafc;
            border-radius: 8px;
        }
        .stat-value { font-size: 32px; font-weight: bold; color: #3b82f6; }
        .stat-label { font-size: 12px; color: #64748b; text-transform: uppercase; margin-top: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Score.Framework}} Compliance Report</h1>
            <div class="meta">
                Tenant ID: {{.TenantID}} |
                Period: {{formatDate .Score.PeriodStart}} - {{formatDate .Score.PeriodEnd}} |
                Generated: {{formatTime .GeneratedAt}}
            </div>
        </div>

        <div class="score-banner">
            <div class="score-circle">
                <div class="score-value">{{formatNumber .Score.OverallScore}}</div>
                <div class="score-label">Compliance Score</div>
            </div>
        </div>

        <div class="content">
            <div class="summary-stats">
                <div class="stat-card">
                    <div class="stat-value">{{.Score.TotalControls}}</div>
                    <div class="stat-label">Total Controls</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value" style="color: #10b981;">{{.Score.PassedControls}}</div>
                    <div class="stat-label">Passed</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value" style="color: #ef4444;">{{.Score.FailedControls}}</div>
                    <div class="stat-label">Failed</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value" style="color: #f59e0b;">{{.Score.SkippedControls}}</div>
                    <div class="stat-label">Skipped</div>
                </div>
            </div>

            <div class="section">
                <h2 class="section-title">Control Assessment</h2>
                <div class="control-grid">
                    {{range $category, $cs := .Score.ControlScores}}
                    <div class="control-card {{$cs.Status}}">
                        <div class="control-name">
                            {{$cs.Category | title}}
                            <span class="badge {{$cs.Status}}">{{$cs.Status}}</span>
                        </div>
                        <div style="font-size: 24px; font-weight: bold;">
                            {{formatNumber $cs.Score}}<span style="font-size: 14px; color: #64748b;">/100</span>
                        </div>
                        <div class="score-bar">
                            <div class="score-fill" style="width: {{$cs.Score}}%; background: {{scoreColor $cs.Score}};"></div>
                        </div>
                        <div style="font-size: 12px; color: #64748b;">
                            Weight: {{formatPercent $cs.Weight}} | {{$cs.FindingsCount}} findings
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>

            {{if .Score.Findings}}
            <div class="section">
                <h2 class="section-title">Compliance Findings</h2>
                <div class="findings-list">
                    {{range .Score.Findings}}
                    <div class="finding {{.Severity}}">
                        <strong>{{.ControlID}}: {{.ControlName}}</strong>
                        <div style="margin-top: 5px;">{{.Description}}</div>
                        {{if .Remediation}}
                        <div style="margin-top: 5px; font-style: italic;">
                            <strong>Remediation:</strong> {{.Remediation}}
                        </div>
                        {{end}}
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}

            {{if .Score.Recommendations}}
            <div class="section">
                <h2 class="section-title">Recommendations</h2>
                <ul style="padding-left: 20px;">
                    {{range .Score.Recommendations}}
                    <li style="margin-bottom: 10px;">{{.}}</li>
                    {{end}}
                </ul>
            </div>
            {{end}}
        </div>

        <div class="footer">
            Generated by OpenPAM Analytics Engine on {{formatTime .GeneratedAt}} |
            Report ID: {{.TenantID}}-{{formatTime .GeneratedAt}}
        </div>
    </div>
</body>
</html>`

	return template.Must(template.New("compliance_report").Funcs(templateFuncs).Parse(tmpl))
}

// GetAnomalyReportTemplate returns the anomaly report HTML template
func GetAnomalyReportTemplate() *template.Template {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Anomaly Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            padding: 20px;
            background: #f5f7fa;
        }
        .container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; overflow: hidden; }
        .header { background: #dc2626; color: white; padding: 30px; }
        .content { padding: 30px; }
        .anomaly-table { width: 100%; border-collapse: collapse; }
        .anomaly-table th, .anomaly-table td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #e1e8ed;
        }
        .anomaly-table th { background: #f8fafc; font-weight: 600; }
        .severity-critical { color: #dc2626; font-weight: bold; }
        .severity-high { color: #ea580c; font-weight: bold; }
        .severity-medium { color: #ca8a04; }
        .severity-low { color: #16a34a; }
        .badge {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 600;
            text-transform: uppercase;
        }
        .badge-open { background: #fef2f2; color: #991b1b; }
        .badge-investigating { background: #fef3c7; color: #92400e; }
        .badge-resolved { background: #d1fae5; color: #065f46; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Security Anomaly Report</h1>
            <div>
                Period: {{formatDate .PeriodStart}} - {{formatDate .PeriodEnd}} |
                Total Anomalies: {{.Count}}
            </div>
        </div>
        <div class="content">
            <table class="anomaly-table">
                <thead>
                    <tr>
                        <th>Type</th>
                        <th>Severity</th>
                        <th>Risk Score</th>
                        <th>Title</th>
                        <th>Detected</th>
                        <th>Status</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Anomalies}}
                    <tr>
                        <td>{{.AnomalyType}}</td>
                        <td class="severity-{{.Severity}}">{{.Severity | title}}</td>
                        <td>{{formatNumber .RiskScore}}</td>
                        <td>{{.Title}}</td>
                        <td>{{formatTime .DetectedAt}}</td>
                        <td><span class="badge badge-{{.Status}}">{{.Status}}</span></td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
    </div>
</body>
</html>`

	return template.Must(template.New("anomaly_report").Funcs(templateFuncs).Parse(tmpl))
}

// GetBaselineReportTemplate returns the baseline report HTML template
func GetBaselineReportTemplate() *template.Template {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Behavioral Baseline Report</title>
    <style>
        body { font-family: -apple-system, sans-serif; padding: 20px; background: #f5f7fa; }
        .container { max-width: 1000px; margin: 0 auto; background: white; border-radius: 8px; }
        .header { background: #3b82f6; color: white; padding: 30px; }
        .content { padding: 30px; }
        .stat-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; }
        .stat-card { padding: 20px; background: #f8fafc; border-radius: 8px; text-align: center; }
        .stat-value { font-size: 36px; font-weight: bold; color: #3b82f6; }
        .stat-label { font-size: 12px; color: #64748b; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Behavioral Baseline Report</h1>
            <div>Generated: {{formatTime .GeneratedAt}}</div>
        </div>
        <div class="content">
            <div class="stat-grid">
                <div class="stat-card">
                    <div class="stat-value">{{.Stats.ActiveBaselines}}</div>
                    <div class="stat-label">Active Baselines</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value">{{.Stats.UniqueUsers}}</div>
                    <div class="stat-label">Users with Baselines</div>
                </div>
                <div class="stat-card">
                    <div class="stat-value">{{formatNumber .Stats.AvgConfidence}}%</div>
                    <div class="stat-label">Avg Confidence</div>
                </div>
            </div>
        </div>
    </div>
</body>
</html>`

	return template.Must(template.New("baseline_report").Funcs(templateFuncs).Parse(tmpl))
}

// GetSummaryReportTemplate returns the summary report HTML template
func GetSummaryReportTemplate() *template.Template {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Security Summary Report</title>
    <style>
        body { font-family: -apple-system, sans-serif; padding: 20px; background: #f5f7fa; }
        .container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 8px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; }
        .content { padding: 30px; }
        .section { margin-bottom: 40px; }
        .section-title { font-size: 20px; font-weight: 600; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Security Summary Report</h1>
            <div>
                Period: {{formatDate .PeriodStart}} - {{formatDate .PeriodEnd}} |
                Generated: {{formatTime .GeneratedAt}}
            </div>
        </div>
        <div class="content">
            <p>Summary report for tenant {{.TenantID}}</p>
        </div>
    </div>
</body>
</html>`

	return template.Must(template.New("summary_report").Funcs(templateFuncs).Parse(tmpl))
}

// Template helper functions

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05 MST")
}

func formatNumber(n float64) string {
	if n == float64(int(n)) {
		return fmt.Sprintf("%.0f", n)
	}
	return fmt.Sprintf("%.1f", n)
}

func formatPercent(n float64) string {
	return fmt.Sprintf("%.0f%%", n*100)
}

func severityColor(severity string) string {
	switch severity {
	case "critical":
		return "#dc2626"
	case "high":
		return "#ea580c"
	case "medium":
		return "#ca8a04"
	case "low":
		return "#16a34a"
	default:
		return "#64748b"
	}
}

func scoreColor(score float64) string {
	switch {
	case score >= 80:
		return "#10b981"
	case score >= 60:
		return "#f59e0b"
	default:
		return "#ef4444"
	}
}
