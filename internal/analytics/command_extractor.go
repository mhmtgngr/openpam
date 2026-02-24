package analytics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/rs/zerolog"
)

// CommandExtractor parses and analyzes commands from session recordings
type CommandExtractor struct {
	repo   Repository
	logger zerolog.Logger
}

// ParsedCommand represents a parsed command with metadata
type ParsedCommand struct {
	Original   string
	Normalized string
	BaseCommand string
	Pattern    string
	Arguments  []string
	RiskLevel  string
	Flags      []string
}

// Dangerous command patterns
var dangerousPatterns = map[string]string{
	"rm -rf":          "critical",
	"rm -r":           "high",
	"rm -f":           "high",
	"mkfs":            "critical",
	"dd if=":          "critical",
	"chmod 000":       "high",
	"chown":           "medium",
	":(){ :|:& };:":   "critical", // Fork bomb
	"wget":            "medium",
	"curl":            "medium",
	"nc -l":           "high",     // Netcat listener
	"ssh-keygen":      "medium",
	"base64 -d":       "high",
	"eval":            "medium",
	"exec":            "medium",
	"kill -9":         "medium",
	"killall":         "high",
	"shutdown":        "critical",
	"reboot":          "high",
	"init 0":          "critical",
	"systemctl stop":  "high",
	"systemctl disable": "high",
	"iptables -F":     "high",
	"crontab":         "medium",
	"at now":          "medium",
	"echo.*>.*\\s*\\/": "high", // Writing to system files
}

// Risk command categories
var riskCategories = map[string]struct {
	commands []string
	risk     string
}{
	"file_deletion": {
		commands: []string{"rm", "del", "rmdir"},
		risk:     "high",
	},
	"privilege_escalation": {
		commands: []string{"sudo", "su", "doas"},
		risk:     "medium",
	},
	"system_modification": {
		commands: []string{"chmod", "chown", "mv", "cp", "dd"},
		risk:     "medium",
	},
	"network_exfiltration": {
		commands: []string{"nc", "netcat", "curl", "wget", "ssh"},
		risk:     "medium",
	},
	"encryption": {
		commands: []string{"openssl", "gpg"},
		risk:     "medium",
	},
}

// NewCommandExtractor creates a new command extractor
func NewCommandExtractor(repo Repository, logger zerolog.Logger) *CommandExtractor {
	return &CommandExtractor{
		repo:   repo,
		logger: logger,
	}
}

// ParseCommand parses a command string and extracts metadata
func (e *CommandExtractor) ParseCommand(command string) *ParsedCommand {
	// Trim whitespace
	original := strings.TrimSpace(command)
	if original == "" {
		return nil
	}

	// Remove shell prompts
	original = e.stripPrompt(original)
	if original == "" {
		return nil
	}

	// Parse into components
	parts := strings.Fields(original)
	if len(parts) == 0 {
		return nil
	}

	baseCommand := parts[0]
	arguments := parts[1:]
	flags := e.extractFlags(arguments)

	// Normalize the command
	normalized := e.normalizeCommand(original)

	// Create pattern for grouping
	pattern := e.createPattern(baseCommand, arguments)

	// Assess risk
	riskLevel := e.assessRiskInternal(baseCommand, original)

	return &ParsedCommand{
		Original:    original,
		Normalized:  normalized,
		BaseCommand: baseCommand,
		Pattern:     pattern,
		Arguments:   arguments,
		RiskLevel:   riskLevel,
		Flags:       flags,
	}
}

// stripPrompt removes shell prompts from command strings
func (e *CommandExtractor) stripPrompt(command string) string {
	// Common shell prompt patterns
	promptPatterns := []string{
		`^[\w\-]+@[\w\-]+:[^\$#]+[\$#]\s+`,
		`^root@[\w\-]+:.*[\$#]\s+`,
		`^\$ `,
		`^# `,
		`^> `,
		`^sudo: `,
	}

	for _, pattern := range promptPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(command) {
			return re.ReplaceAllString(command, "")
		}
	}

	return command
}

// normalizeCommand creates a normalized version for comparison
func (e *CommandExtractor) normalizeCommand(command string) string {
	// Convert to lowercase for comparison
	normalized := strings.ToLower(strings.TrimSpace(command))

	// Remove variable substitutions
	re := regexp.MustCompile(`\$\{?\w+\}?`)
	normalized = re.ReplaceAllString(normalized, "$VAR")

	// Remove quoted strings
	re = regexp.MustCompile(`["'][^"']*["']`)
	normalized = re.ReplaceAllString(normalized, "\"ARG\"")

	// Collapse multiple spaces
	re = regexp.MustCompile(`\s+`)
	normalized = re.ReplaceAllString(normalized, " ")

	return normalized
}

// createPattern creates a pattern for command grouping
func (e *CommandExtractor) createPattern(baseCommand string, arguments []string) string {
	if len(arguments) == 0 {
		return baseCommand
	}

	// Build pattern with flags but without specific values
	var patternParts []string
	patternParts = append(patternParts, baseCommand)

	for _, arg := range arguments {
		if strings.HasPrefix(arg, "-") {
			patternParts = append(patternParts, arg)
		} else if isFilePath(arg) {
			patternParts = append(patternParts, "<FILE>")
		} else {
			patternParts = append(patternParts, "<ARG>")
		}
	}

	return strings.Join(patternParts, " ")
}

// extractFlags extracts command flags
func (e *CommandExtractor) extractFlags(arguments []string) []string {
	var flags []string
	for _, arg := range arguments {
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
		}
	}
	return flags
}

// AssessRisk assesses the risk level of a command
func (e *CommandExtractor) AssessRisk(command *ParsedCommand) string {
	if command == nil {
		return string(RiskLevelLow)
	}
	return command.RiskLevel
}

// assessRiskInternal determines the risk level of a command
func (e *CommandExtractor) assessRiskInternal(baseCommand, original string) string {
	originalLower := strings.ToLower(original)

	// Check against dangerous patterns in order of specificity (longer patterns first)
	// This ensures "rm -rf" is checked before "rm -r"
	dangerousPatternsOrdered := []struct {
		pattern string
		risk    string
	}{
		{":(){ :|:& };:", "critical"},         // Fork bomb - most specific
		{"rm -rf", "critical"},                // Recursive force delete
		{"rm -r", "high"},                     // Recursive delete
		{"rm -f", "high"},                     // Force delete
		{"mkfs", "critical"},                  // Filesystem creation
		{"dd if=", "critical"},                // Direct disk write
		{"chmod 000", "high"},                 // Remove all permissions
		{"chown", "medium"},                   // Change owner
		{"nc -l", "high"},                     // Netcat listener
		{"base64 -d", "high"},                 // Base64 decode
		{"killall", "high"},                   // Kill all
		{"shutdown", "critical"},              // System shutdown
		{"reboot", "high"},                    // System reboot
		{"init 0", "critical"},                // System halt
		{"systemctl stop", "high"},            // Stop service
		{"systemctl disable", "high"},         // Disable service
		{"iptables -f", "high"},               // Flush iptables (lowercase for matching)
		{"echo.*>.*\\s*\\/", "high"},          // Writing to system files
	}

	// Check for pipe chains FIRST - these are most dangerous
	// Commands like "curl | bash" should be high risk even if curl alone is medium
	if strings.Contains(original, "|") {
		parts := strings.Split(original, "|")
		if len(parts) > 1 {
			// Any pipe chain involving network commands or eval is high risk
			for _, part := range parts {
				partLower := strings.ToLower(strings.TrimSpace(part))
				// Check if this part contains a network command or eval
				if strings.Contains(partLower, "curl") ||
					strings.Contains(partLower, "wget") ||
					strings.Contains(partLower, "eval") ||
					strings.Contains(partLower, "sh") ||
					strings.Contains(partLower, "bash") {
					return string(RiskLevelHigh)
				}
			}
		}
	}

	// Then check specific dangerous patterns
	for _, dp := range dangerousPatternsOrdered {
		if strings.Contains(originalLower, dp.pattern) {
			return dp.risk
		}
	}

	// Check for specific dangerous combinations BEFORE checking risk categories
	// This ensures combinations like "chmod +x file && ./file" are caught
	if e.hasDangerousCombination(originalLower) {
		return string(RiskLevelHigh)
	}

	// Check general risk patterns (medium risk commands)
	mediumRiskPatterns := []struct {
		pattern string
		risk    string
	}{
		{"wget", "medium"},
		{"curl", "medium"},
		{"ssh-keygen", "medium"},
		{"eval", "medium"},
		{"exec", "medium"},
		{"kill -9", "medium"},
		{"crontab", "medium"},
		{"at now", "medium"},
	}

	for _, dp := range mediumRiskPatterns {
		if strings.Contains(originalLower, dp.pattern) {
			return dp.risk
		}
	}

	// Check against risk categories (after combinations check)
	for _, category := range riskCategories {
		for _, cmd := range category.commands {
			if baseCommand == cmd {
				return category.risk
			}
		}
	}

	// Default: low risk
	return string(RiskLevelLow)
}

// hasDangerousCombination checks for dangerous command combinations
func (e *CommandExtractor) hasDangerousCombination(command string) bool {
	dangerousCombos := []string{
		"chmod +x",
		"curl | bash",
		"wget | bash",
		"curl | sh",
		"wget | sh",
		"eval $(",
		"$(curl",
		"$(wget",
		"&& rm",
		"; rm",
	}

	cmdLower := strings.ToLower(command)
	for _, combo := range dangerousCombos {
		if strings.Contains(cmdLower, combo) {
			return true
		}
	}

	return false
}

// isFilePath checks if a string looks like a file path
func isFilePath(s string) bool {
	return strings.Contains(s, "/") || strings.Contains(s, "\\")
}

// GetCommandHash generates a hash for command deduplication
func (e *CommandExtractor) GetCommandHash(command string) string {
	hash := sha256.Sum256([]byte(command))
	return hex.EncodeToString(hash[:])
}

// IsFileOperation checks if command is a file operation
func (e *CommandExtractor) IsFileOperation(baseCommand string) bool {
	fileOps := []string{"rm", "mv", "cp", "touch", "mkdir", "rmdir", "chmod", "chown", "ln"}
	for _, op := range fileOps {
		if baseCommand == op {
			return true
		}
	}
	return false
}

// IsNetworkOperation checks if command is a network operation
func (e *CommandExtractor) IsNetworkOperation(baseCommand string) bool {
	netOps := []string{"curl", "wget", "nc", "netcat", "ssh", "scp", "rsync", "ftp", "telnet"}
	for _, op := range netOps {
		if baseCommand == op {
			return true
		}
	}
	return false
}

// IsPrivilegeEscalation checks if command attempts privilege escalation
func (e *CommandExtractor) IsPrivilegeEscalation(command string) bool {
	privEscPatterns := []string{"sudo ", "su ", "doas ", "sudo -i"}
	cmdLower := strings.ToLower(command)
	for _, pattern := range privEscPatterns {
		if strings.HasPrefix(cmdLower, pattern) {
			return true
		}
	}
	return false
}

// ExtractCommandsFromRecording extracts commands from a session recording
func (e *CommandExtractor) ExtractCommandsFromRecording(recordingData []byte) []string {
	var commands []string

	// Convert bytes to string
	content := string(recordingData)

	// Split by lines
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Strip shell prompt
		trimmed = e.stripPrompt(trimmed)
		if trimmed == "" {
			continue
		}

		// Skip obvious non-command lines
		if e.isNonCommandLine(trimmed) {
			continue
		}

		commands = append(commands, trimmed)
	}

	return commands
}

// isNonCommandLine checks if a line is not a command
func (e *CommandExtractor) isNonCommandLine(line string) bool {
	// Skip comments
	if strings.HasPrefix(line, "#") {
		return true
	}

	// Skip output lines (start with typical output prefixes)
	outputPrefixes := []string{"[", "]", "(", ")", "=", "+", "-", "*", "│", "└", "├", "→"}
	for _, prefix := range outputPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	// Skip lines that are all non-command characters (likely output)
	for _, r := range line {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return false
		}
	}
	return len(line) > 0
}

// BatchParseCommands parses multiple commands
func (e *CommandExtractor) BatchParseCommands(commands []string) []*ParsedCommand {
	var parsed []*ParsedCommand

	for _, cmd := range commands {
		if p := e.ParseCommand(cmd); p != nil {
			parsed = append(parsed, p)
		}
	}

	return parsed
}

// GetCommandStatistics returns statistics about parsed commands
func (e *CommandExtractor) GetCommandStatistics(commands []*ParsedCommand) map[string]interface{} {
	stats := make(map[string]interface{})

	riskCounts := make(map[string]int)
	baseCommandCounts := make(map[string]int)
	var highRiskCount int

	for _, cmd := range commands {
		riskCounts[cmd.RiskLevel]++
		baseCommandCounts[cmd.BaseCommand]++

		if cmd.RiskLevel == string(RiskLevelHigh) || cmd.RiskLevel == string(RiskLevelCritical) {
			highRiskCount++
		}
	}

	stats["total_commands"] = len(commands)
	stats["risk_distribution"] = riskCounts
	stats["top_commands"] = baseCommandCounts
	stats["high_risk_count"] = highRiskCount

	return stats
}

// CreateCommandSignature creates a unique signature for a command
func (e *CommandExtractor) CreateCommandSignature(command *ParsedCommand) string {
	parts := []string{
		command.BaseCommand,
		command.Pattern,
	}
	return strings.Join(parts, ":")
}

// ShouldBlockCommand determines if a command should be blocked
func (e *CommandExtractor) ShouldBlockCommand(command *ParsedCommand) bool {
	if command == nil {
		return false
	}

	// Block critical risk commands
	if command.RiskLevel == string(RiskLevelCritical) {
		return true
	}

	// Block specific dangerous patterns
	dangerousCriticalPatterns := []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs",
		":(){ :|:& };:",
		"dd if=/dev/zero",
		"dd if=/dev/random",
	}

	cmdLower := strings.ToLower(command.Original)
	for _, pattern := range dangerousCriticalPatterns {
		if strings.Contains(cmdLower, pattern) {
			return true
		}
	}

	return false
}

// SanitizeCommand returns a sanitized version of command for logging
func (e *CommandExtractor) SanitizeCommand(command string) string {
	// Remove sensitive arguments like passwords, tokens, etc.
	sanitized := command

	// Remove password flags
	re := regexp.MustCompile(`-p\s*\S+`)
	sanitized = re.ReplaceAllString(sanitized, "-p ****")

	// Remove token arguments
	re = regexp.MustCompile(`token=\S+`)
	sanitized = re.ReplaceAllString(sanitized, "token=****")

	// Remove API key arguments
	re = regexp.MustCompile(`[aA][pP][iI]_[kK][eE][yY]=\S+`)
	sanitized = re.ReplaceAllString(sanitized, "api_key=****")

	return sanitized
}

// GetCommandCategory categorizes a command
func (e *CommandExtractor) GetCommandCategory(baseCommand string) string {
	categories := map[string][]string{
		"file_management": {"ls", "cd", "pwd", "mkdir", "rmdir", "touch", "find", "locate"},
		"file_modification": {"cp", "mv", "rm", "chmod", "chown", "ln"},
		"text_processing": {"cat", "less", "more", "head", "tail", "grep", "sed", "awk", "sort", "uniq"},
		"archive": {"tar", "zip", "unzip", "gzip", "gunzip"},
		"network": {"curl", "wget", "nc", "netcat", "ping", "traceroute", "nslookup", "dig"},
		"remote": {"ssh", "scp", "rsync", "telnet", "ftp"},
		"system": {"systemctl", "service", "ps", "top", "kill", "killall", "shutdown", "reboot"},
		"user_management": {"useradd", "userdel", "usermod", "passwd", "groupadd"},
		"package": {"apt", "apt-get", "yum", "dnf", "pacman", "rpm"},
		"database": {"mysql", "psql", "mongo", "redis-cli"},
	}

	for category, commands := range categories {
		for _, cmd := range commands {
			if baseCommand == cmd {
				return category
			}
		}
	}

	return "other"
}

// GenerateCommandReport generates a human-readable command report
func (e *CommandExtractor) GenerateCommandReport(commands []*ParsedCommand) string {
	var report strings.Builder

	report.WriteString(fmt.Sprintf("Command Analysis Report\n"))
	report.WriteString(fmt.Sprintf("======================\n\n"))
	report.WriteString(fmt.Sprintf("Total Commands: %d\n\n", len(commands)))

	// Risk distribution
	riskCounts := make(map[string]int)
	for _, cmd := range commands {
		riskCounts[cmd.RiskLevel]++
	}

	report.WriteString("Risk Distribution:\n")
	for risk, count := range riskCounts {
		report.WriteString(fmt.Sprintf("  %s: %d\n", strings.ToUpper(risk), count))
	}

	report.WriteString("\nTop Commands:\n")
	baseCommandCounts := make(map[string]int)
	for _, cmd := range commands {
		baseCommandCounts[cmd.BaseCommand]++
	}

	// Sort and display top 10
	count := 0
	for cmd, cmdCount := range baseCommandCounts {
		if count >= 10 {
			break
		}
		report.WriteString(fmt.Sprintf("  %s: %d\n", cmd, cmdCount))
		count++
	}

	return report.String()
}
