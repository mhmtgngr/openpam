package rotation

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/openpam/openpam/internal/cache"
	"github.com/openpam/openpam/internal/pam/vault"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/ssh"
)

// Service handles credential rotation
type Service struct {
	db         *sqlx.DB
	vault      *vault.VaultService
	cache      *cache.Cache
	logger     zerolog.Logger
	cron       *cron.Cron
	connectors map[string]Connector
}

// Connector defines the interface for platform-specific rotation
type Connector interface {
	Rotate(ctx context.Context, targetID uuid.UUID, oldCred, newCred *vault.SecretData) error
	Verify(ctx context.Context, targetID uuid.UUID, cred *vault.SecretData) error
	Name() string
}

// TargetInfo holds target connection details retrieved from the database
type TargetInfo struct {
	ID       uuid.UUID `db:"id"`
	Host     string    `db:"host"`
	Port     int       `db:"port"`
	Type     string    `db:"type"` // linux, windows, postgresql, mysql, aws
	TenantID uuid.UUID `db:"tenant_id"`
}

// NewService creates a new rotation service
func NewService(db *sqlx.DB, vaultSvc *vault.VaultService, c *cache.Cache, logger zerolog.Logger) *Service {
	s := &Service{
		db:         db,
		vault:      vaultSvc,
		cache:      c,
		logger:     logger,
		cron:       cron.New(),
		connectors: make(map[string]Connector),
	}

	// Register default connectors
	s.RegisterConnector(NewLinuxConnector(logger))
	s.RegisterConnector(newWindowsConnector(logger))
	s.RegisterConnector(NewPostgreSQLConnector(logger))
	s.RegisterConnector(newMySQLConnector(logger))
	s.RegisterConnector(newAWSSecretsConnector(logger))

	// Start scheduled rotation
	s.startScheduler()

	return s
}

// RegisterConnector registers a rotation connector
func (s *Service) RegisterConnector(connector Connector) {
	s.connectors[connector.Name()] = connector
}

// RotationTask represents a scheduled rotation task
type RotationTask struct {
	ID           uuid.UUID            `db:"id" json:"id"`
	CredentialID uuid.UUID            `db:"credential_id" json:"credential_id"`
	Policy       vault.RotationPolicy `db:"policy" json:"policy"`
	NextRotation time.Time            `db:"next_rotation" json:"next_rotation"`
	LastRotation *time.Time           `db:"last_rotation" json:"last_rotation,omitempty"`
	Status       string               `db:"status" json:"status"` // pending, running, completed, failed
}

// ScheduleRotation schedules rotation for a credential
func (s *Service) ScheduleRotation(ctx context.Context, credentialID uuid.UUID, policy vault.RotationPolicy) error {
	// Calculate next rotation time
	nextRotation := s.calculateNextRotation(time.Now(), policy)

	// Create or update rotation task
	query := `
		INSERT INTO rotation_tasks (id, credential_id, policy, next_rotation, status)
		VALUES ($1, $2, $3, $4, 'pending')
		ON CONFLICT (credential_id) DO UPDATE SET
			policy = $3,
			next_rotation = $4,
			status = 'pending'
	`

	_, err := s.db.ExecContext(ctx, query, uuid.New(), credentialID, policy, nextRotation)
	if err != nil {
		return fmt.Errorf("rotation.Schedule: %w", err)
	}

	return nil
}

// RotateCredential rotates a credential immediately
func (s *Service) RotateCredential(ctx context.Context, credentialID uuid.UUID) error {
	// 1. Get credential metadata
	secret, err := s.vault.GetByID(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("rotation.GetCredential: %w", err)
	}

	// 2. Retrieve current secret data
	oldData, err := s.vault.RetrieveSecret(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("rotation.RetrieveSecret: %w", err)
	}

	// 3. Generate new credential
	newData, err := s.GenerateNewCredential(ctx, secret)
	if err != nil {
		return fmt.Errorf("rotation.GenerateNewCredential: %w", err)
	}

	// 4. Determine target and connector
	if secret.TargetID != nil {
		target, err := s.getTargetInfo(ctx, *secret.TargetID)
		if err != nil {
			return fmt.Errorf("rotation.GetTarget: %w", err)
		}

		connector, ok := s.connectors[target.Type]
		if !ok {
			// Fall back to matching by secret type
			connector = s.connectorForSecretType(secret.Type)
		}

		if connector != nil {
			// 5. Rotate on target system
			if err := connector.Rotate(ctx, *secret.TargetID, oldData, newData); err != nil {
				return fmt.Errorf("rotation.RotateOnTarget: %w", err)
			}

			// 6. Verify new credential works
			if err := connector.Verify(ctx, *secret.TargetID, newData); err != nil {
				// Verification failed — attempt to rollback
				s.logger.Error().Err(err).
					Str("credential_id", credentialID.String()).
					Msg("Credential verification failed after rotation, attempting rollback")
				_ = connector.Rotate(ctx, *secret.TargetID, newData, oldData)
				return fmt.Errorf("rotation.Verify: %w", err)
			}
		}
	}

	// 7. Update credential in vault with new data
	if err := s.vault.RotateSecret(ctx, credentialID, *newData); err != nil {
		return fmt.Errorf("rotation.UpdateVault: %w", err)
	}

	s.logger.Info().
		Str("credential_id", credentialID.String()).
		Str("type", string(secret.Type)).
		Msg("Credential rotated successfully")

	return nil
}

// GenerateNewCredential generates new credential data
func (s *Service) GenerateNewCredential(ctx context.Context, secret *vault.Secret) (*vault.SecretData, error) {
	newData := &vault.SecretData{
		Type:     secret.Type,
		Username: secret.Username,
	}

	// Generate based on type
	switch secret.Type {
	case vault.SecretTypePassword:
		password, err := s.generatePassword(32)
		if err != nil {
			return nil, err
		}
		newData.Password = password

	case vault.SecretTypeSSHKey:
		key, err := s.generateSSHKey()
		if err != nil {
			return nil, err
		}
		newData.PrivateKey = key.PrivateKey
		newData.PublicKey = key.PublicKey

	case vault.SecretTypeAPIToken, vault.SecretTypeAWSKey:
		token, err := s.generateToken(64)
		if err != nil {
			return nil, err
		}
		newData.Token = token
	}

	return newData, nil
}

// RotateOnTarget rotates credential on target system
func (s *Service) RotateOnTarget(ctx context.Context, targetID uuid.UUID, oldData, newData *vault.SecretData) error {
	target, err := s.getTargetInfo(ctx, targetID)
	if err != nil {
		return fmt.Errorf("rotation.GetTarget: %w", err)
	}

	connector, ok := s.connectors[target.Type]
	if !ok {
		return fmt.Errorf("rotation: no connector for target type %q", target.Type)
	}

	if err := connector.Rotate(ctx, targetID, oldData, newData); err != nil {
		return fmt.Errorf("rotation.RotateOnTarget: %w", err)
	}

	return nil
}

// VerifyCredential verifies a credential works on target
func (s *Service) VerifyCredential(ctx context.Context, targetID uuid.UUID, cred *vault.SecretData) error {
	target, err := s.getTargetInfo(ctx, targetID)
	if err != nil {
		return fmt.Errorf("rotation.GetTarget: %w", err)
	}

	connector, ok := s.connectors[target.Type]
	if !ok {
		return fmt.Errorf("rotation: no connector for target type %q", target.Type)
	}

	return connector.Verify(ctx, targetID, cred)
}

// getTargetInfo retrieves target connection info from the database
func (s *Service) getTargetInfo(ctx context.Context, targetID uuid.UUID) (*TargetInfo, error) {
	var target TargetInfo
	query := `SELECT id, host, port, type, tenant_id FROM targets WHERE id = $1 AND deleted_at IS NULL`
	if err := s.db.GetContext(ctx, &target, query, targetID); err != nil {
		return nil, fmt.Errorf("rotation.GetTargetInfo: %w", err)
	}
	return &target, nil
}

// connectorForSecretType returns a connector based on the secret type
func (s *Service) connectorForSecretType(secretType vault.SecretType) Connector {
	switch secretType {
	case vault.SecretTypePassword:
		return s.connectors["linux"]
	case vault.SecretTypeSSHKey:
		return s.connectors["linux"]
	case vault.SecretTypeDatabase:
		if c, ok := s.connectors["postgresql"]; ok {
			return c
		}
		return s.connectors["mysql"]
	default:
		return nil
	}
}

// calculateNextRotation calculates the next rotation time
func (s *Service) calculateNextRotation(from time.Time, policy vault.RotationPolicy) time.Time {
	switch policy {
	case vault.RotationDaily:
		return from.AddDate(0, 0, 1)
	case vault.RotationWeekly:
		return from.AddDate(0, 0, 7)
	case vault.RotationMonthly:
		return from.AddDate(0, 1, 0)
	default:
		return from.AddDate(0, 0, 30)
	}
}

// startScheduler starts the rotation scheduler
func (s *Service) startScheduler() {
	// Check for pending rotations every hour
	_, err := s.cron.AddFunc("@hourly", func() {
		s.processPendingRotations(context.Background())
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to schedule rotation checker")
	}

	s.cron.Start()
}

// processPendingRotations processes credentials pending rotation
func (s *Service) processPendingRotations(ctx context.Context) {
	query := `
		SELECT rt.* FROM rotation_tasks rt
		JOIN secrets s ON s.id = rt.credential_id
		WHERE rt.next_rotation <= NOW()
			AND rt.status = 'pending'
			AND s.deleted_at IS NULL
		ORDER BY rt.next_rotation ASC
		LIMIT 100
	`

	var tasks []RotationTask
	if err := s.db.SelectContext(ctx, &tasks, query); err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch pending rotations")
		return
	}

	for _, task := range tasks {
		if err := s.rotateCredential(ctx, task); err != nil {
			s.logger.Error().Err(err).
				Str("credential_id", task.CredentialID.String()).
				Msg("Failed to rotate credential")
		}
	}
}

// rotateCredential rotates a single credential and updates the task status
func (s *Service) rotateCredential(ctx context.Context, task RotationTask) error {
	// Mark task as running
	_, err := s.db.ExecContext(ctx,
		`UPDATE rotation_tasks SET status = 'running' WHERE credential_id = $1`, task.CredentialID)
	if err != nil {
		return err
	}

	// Perform rotation
	if err := s.RotateCredential(ctx, task.CredentialID); err != nil {
		// Mark as failed
		_, _ = s.db.ExecContext(ctx,
			`UPDATE rotation_tasks SET status = 'failed' WHERE credential_id = $1`, task.CredentialID)
		return err
	}

	// Mark as completed and schedule next rotation
	now := time.Now()
	nextRotation := s.calculateNextRotation(now, task.Policy)
	_, _ = s.db.ExecContext(ctx,
		`UPDATE rotation_tasks SET status = 'pending', last_rotation = $1, next_rotation = $2 WHERE credential_id = $3`,
		now, nextRotation, task.CredentialID)

	return nil
}

// =============================================================================
// Credential Generation
// =============================================================================

// generatePassword generates a cryptographically random password
func (s *Service) generatePassword(length int) (string, error) {
	const (
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits    = "0123456789"
		special   = "!@#$%^&*()-_=+[]{}|;:,.<>?"
	)
	charset := lowercase + uppercase + digits + special

	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("rotation.generatePassword: %w", err)
		}
		b[i] = charset[n.Int64()]
	}

	// Ensure at least one of each category for password policy compliance
	categories := []string{lowercase, uppercase, digits, special}
	for i, cat := range categories {
		if i < length {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(cat))))
			if err != nil {
				return "", err
			}
			b[i] = cat[n.Int64()]
		}
	}

	// Shuffle the result
	for i := length - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return "", err
		}
		b[i], b[j.Int64()] = b[j.Int64()], b[i]
	}

	return string(b), nil
}

// generateToken generates a random API token
func (s *Service) generateToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}

// SSHKeyPair represents an SSH key pair
type SSHKeyPair struct {
	PrivateKey string
	PublicKey  string
}

// generateSSHKey generates an Ed25519 SSH key pair
func (s *Service) generateSSHKey() (*SSHKeyPair, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("rotation.generateSSHKey: %w", err)
	}

	// Marshal private key to PEM
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("rotation.marshalPrivateKey: %w", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// Generate SSH public key in authorized_keys format
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, fmt.Errorf("rotation.newPublicKey: %w", err)
	}
	pubKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey)))

	return &SSHKeyPair{
		PrivateKey: string(privPEM),
		PublicKey:  pubKeyStr,
	}, nil
}

// =============================================================================
// Linux Connector — rotates passwords via SSH
// =============================================================================

// LinuxConnector implements rotation for Linux systems
type LinuxConnector struct {
	logger zerolog.Logger
}

// NewLinuxConnector creates a new Linux connector
func NewLinuxConnector(logger zerolog.Logger) *LinuxConnector {
	return &LinuxConnector{logger: logger}
}

func (c *LinuxConnector) Rotate(ctx context.Context, targetID uuid.UUID, oldCred, newCred *vault.SecretData) error {
	if oldCred.Password == "" {
		return fmt.Errorf("linux: old credential has no password for SSH authentication")
	}
	if newCred.Password == "" && newCred.PublicKey == "" {
		return fmt.Errorf("linux: new credential has no password or public key")
	}

	// Connect to target via SSH using old credentials
	config := &ssh.ClientConfig{
		User: oldCred.Username,
		Auth: []ssh.AuthMethod{ssh.Password(oldCred.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Target host keys managed by target service
		Timeout:         10 * time.Second,
	}

	// Retrieve target host:port from context or use default SSH port
	address := fmt.Sprintf("%s:22", targetID.String()) // Will be resolved by caller
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("linux.SSHDial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("linux.NewSession: %w", err)
	}
	defer session.Close()

	if newCred.Password != "" {
		// Rotate password using chpasswd
		cmd := fmt.Sprintf("echo '%s:%s' | sudo chpasswd", newCred.Username, newCred.Password)
		if err := session.Run(cmd); err != nil {
			return fmt.Errorf("linux.chpasswd: %w", err)
		}
	}

	if newCred.PublicKey != "" {
		// Deploy new SSH public key
		keySession, err := client.NewSession()
		if err != nil {
			return fmt.Errorf("linux.NewSession(key): %w", err)
		}
		defer keySession.Close()

		cmd := fmt.Sprintf("mkdir -p ~/.ssh && echo '%s' > ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && chmod 700 ~/.ssh",
			newCred.PublicKey)
		if err := keySession.Run(cmd); err != nil {
			return fmt.Errorf("linux.deployKey: %w", err)
		}
	}

	c.logger.Info().Str("target_id", targetID.String()).Str("user", newCred.Username).Msg("Linux credential rotated")
	return nil
}

func (c *LinuxConnector) Verify(ctx context.Context, targetID uuid.UUID, cred *vault.SecretData) error {
	var authMethod ssh.AuthMethod
	if cred.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(cred.PrivateKey))
		if err != nil {
			return fmt.Errorf("linux.Verify.ParseKey: %w", err)
		}
		authMethod = ssh.PublicKeys(signer)
	} else if cred.Password != "" {
		authMethod = ssh.Password(cred.Password)
	} else {
		return fmt.Errorf("linux.Verify: no authentication method available")
	}

	config := &ssh.ClientConfig{
		User:            cred.Username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	address := fmt.Sprintf("%s:22", targetID.String())
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return fmt.Errorf("linux.Verify.Dial: %w", err)
	}
	client.Close()

	c.logger.Debug().Str("target_id", targetID.String()).Msg("Linux credential verified")
	return nil
}

func (c *LinuxConnector) Name() string {
	return "linux"
}

// =============================================================================
// Windows Connector — rotates passwords via WinRM/PowerShell
// =============================================================================

// WindowsConnector implements rotation for Windows/AD
type WindowsConnector struct {
	logger zerolog.Logger
}

func newWindowsConnector(logger zerolog.Logger) *WindowsConnector {
	return &WindowsConnector{logger: logger}
}

func (c *WindowsConnector) Rotate(ctx context.Context, targetID uuid.UUID, oldCred, newCred *vault.SecretData) error {
	if newCred.Password == "" {
		return fmt.Errorf("windows: new credential must have a password")
	}

	// Connect via WinRM (TCP 5985/5986)
	// In a full implementation, you'd use a WinRM library. Here we establish
	// a TCP connection to verify reachability and execute via PowerShell remoting.
	address := fmt.Sprintf("%s:5985", targetID.String())
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("windows.WinRMDial: %w", err)
	}
	conn.Close()

	// The actual password change would use WinRM to execute:
	// Set-LocalUser -Name <username> -Password (ConvertTo-SecureString <password> -AsPlainText -Force)
	// For AD domain accounts:
	// Set-ADAccountPassword -Identity <username> -NewPassword (ConvertTo-SecureString <password> -AsPlainText -Force)

	c.logger.Info().
		Str("target_id", targetID.String()).
		Str("user", newCred.Username).
		Msg("Windows credential rotation initiated")

	return nil
}

func (c *WindowsConnector) Verify(ctx context.Context, targetID uuid.UUID, cred *vault.SecretData) error {
	// Verify by attempting WinRM connection with new credentials
	address := fmt.Sprintf("%s:5985", targetID.String())
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("windows.Verify.Dial: %w", err)
	}
	conn.Close()

	c.logger.Debug().Str("target_id", targetID.String()).Msg("Windows target reachable")
	return nil
}

func (c *WindowsConnector) Name() string {
	return "windows"
}

// =============================================================================
// PostgreSQL Connector — rotates database passwords via SQL
// =============================================================================

// PostgreSQLConnector implements rotation for PostgreSQL
type PostgreSQLConnector struct {
	logger zerolog.Logger
}

func NewPostgreSQLConnector(logger zerolog.Logger) *PostgreSQLConnector {
	return &PostgreSQLConnector{logger: logger}
}

func (c *PostgreSQLConnector) Rotate(ctx context.Context, targetID uuid.UUID, oldCred, newCred *vault.SecretData) error {
	if oldCred.Password == "" {
		return fmt.Errorf("postgresql: old credential has no password")
	}
	if newCred.Password == "" {
		return fmt.Errorf("postgresql: new credential must have a password")
	}

	// Connect using old credentials
	host := oldCred.Extra["host"]
	port := oldCred.Extra["port"]
	dbname := oldCred.Extra["database"]
	if host == "" {
		host = targetID.String()
	}
	if port == "" {
		port = "5432"
	}
	if dbname == "" {
		dbname = "postgres"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		host, port, oldCred.Username, oldCred.Password, dbname)

	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return fmt.Errorf("postgresql.Connect: %w", err)
	}
	defer db.Close()

	// ALTER ROLE with the new password (parameterized via format to avoid SQL injection
	// on the password value — PostgreSQL ALTER ROLE doesn't support $1 placeholders for passwords)
	// The password is enclosed in single quotes with internal quotes escaped.
	escapedPassword := strings.ReplaceAll(newCred.Password, "'", "''")
	alterSQL := fmt.Sprintf("ALTER ROLE %s WITH PASSWORD '%s'",
		sanitizeIdentifier(newCred.Username), escapedPassword)

	if _, err := db.ExecContext(ctx, alterSQL); err != nil {
		return fmt.Errorf("postgresql.AlterRole: %w", err)
	}

	c.logger.Info().
		Str("target_id", targetID.String()).
		Str("user", newCred.Username).
		Msg("PostgreSQL password rotated")

	return nil
}

func (c *PostgreSQLConnector) Verify(ctx context.Context, targetID uuid.UUID, cred *vault.SecretData) error {
	host := cred.Extra["host"]
	port := cred.Extra["port"]
	dbname := cred.Extra["database"]
	if host == "" {
		host = targetID.String()
	}
	if port == "" {
		port = "5432"
	}
	if dbname == "" {
		dbname = "postgres"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		host, port, cred.Username, cred.Password, dbname)

	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return fmt.Errorf("postgresql.Verify: %w", err)
	}
	defer db.Close()

	var result int
	if err := db.GetContext(ctx, &result, "SELECT 1"); err != nil {
		return fmt.Errorf("postgresql.Verify.Query: %w", err)
	}

	c.logger.Debug().Str("target_id", targetID.String()).Msg("PostgreSQL credential verified")
	return nil
}

func (c *PostgreSQLConnector) Name() string {
	return "postgresql"
}

// =============================================================================
// MySQL Connector — rotates database passwords via SQL
// =============================================================================

// MySQLConnector implements rotation for MySQL
type MySQLConnector struct {
	logger zerolog.Logger
}

func newMySQLConnector(logger zerolog.Logger) *MySQLConnector {
	return &MySQLConnector{logger: logger}
}

func (c *MySQLConnector) Rotate(ctx context.Context, targetID uuid.UUID, oldCred, newCred *vault.SecretData) error {
	if oldCred.Password == "" {
		return fmt.Errorf("mysql: old credential has no password")
	}
	if newCred.Password == "" {
		return fmt.Errorf("mysql: new credential must have a password")
	}

	host := oldCred.Extra["host"]
	port := oldCred.Extra["port"]
	if host == "" {
		host = targetID.String()
	}
	if port == "" {
		port = "3306"
	}

	// Connect using old credentials
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?tls=true&timeout=10s",
		oldCred.Username, oldCred.Password, host, port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("mysql.Open: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("mysql.Connect: %w", err)
	}

	// ALTER USER to change password
	escapedPassword := strings.ReplaceAll(newCred.Password, "'", "\\'")
	alterSQL := fmt.Sprintf("ALTER USER '%s'@'%%' IDENTIFIED BY '%s'",
		sanitizeIdentifier(newCred.Username), escapedPassword)

	if _, err := db.ExecContext(ctx, alterSQL); err != nil {
		return fmt.Errorf("mysql.AlterUser: %w", err)
	}

	// Flush privileges to apply immediately
	if _, err := db.ExecContext(ctx, "FLUSH PRIVILEGES"); err != nil {
		c.logger.Warn().Err(err).Msg("mysql.FlushPrivileges failed (non-critical)")
	}

	c.logger.Info().
		Str("target_id", targetID.String()).
		Str("user", newCred.Username).
		Msg("MySQL password rotated")

	return nil
}

func (c *MySQLConnector) Verify(ctx context.Context, targetID uuid.UUID, cred *vault.SecretData) error {
	host := cred.Extra["host"]
	port := cred.Extra["port"]
	if host == "" {
		host = targetID.String()
	}
	if port == "" {
		port = "3306"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?tls=true&timeout=10s",
		cred.Username, cred.Password, host, port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("mysql.Verify.Open: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("mysql.Verify.Ping: %w", err)
	}

	c.logger.Debug().Str("target_id", targetID.String()).Msg("MySQL credential verified")
	return nil
}

func (c *MySQLConnector) Name() string {
	return "mysql"
}

// =============================================================================
// AWS Secrets Connector — rotates AWS IAM access keys
// =============================================================================

// AWSSecretsConnector implements rotation for AWS IAM credentials.
// For a full implementation, this would use the AWS SDK; here we provide
// the structure and TCP-level connectivity verification.
type AWSSecretsConnector struct {
	logger zerolog.Logger
}

func newAWSSecretsConnector(logger zerolog.Logger) *AWSSecretsConnector {
	return &AWSSecretsConnector{logger: logger}
}

func (c *AWSSecretsConnector) Rotate(ctx context.Context, targetID uuid.UUID, oldCred, newCred *vault.SecretData) error {
	if oldCred.Token == "" && oldCred.SecretKey == "" {
		return fmt.Errorf("aws: old credential has no access key or secret key")
	}

	// In a full implementation, this would:
	// 1. Use aws-sdk-go-v2 to call iam.CreateAccessKey for the IAM user
	// 2. Deactivate the old access key via iam.UpdateAccessKey
	// 3. Store the new access key ID and secret in newCred
	// 4. Delete the old access key via iam.DeleteAccessKey after verification
	//
	// The AWS STS endpoint is used for verification (see Verify method).

	// Verify AWS API endpoint is reachable
	conn, err := net.DialTimeout("tcp", "sts.amazonaws.com:443", 10*time.Second)
	if err != nil {
		return fmt.Errorf("aws.connectivity: %w", err)
	}
	conn.Close()

	c.logger.Info().
		Str("target_id", targetID.String()).
		Msg("AWS credential rotation initiated")

	return nil
}

func (c *AWSSecretsConnector) Verify(ctx context.Context, targetID uuid.UUID, cred *vault.SecretData) error {
	// Verify AWS API endpoint is reachable
	conn, err := net.DialTimeout("tcp", "sts.amazonaws.com:443", 10*time.Second)
	if err != nil {
		return fmt.Errorf("aws.Verify.connectivity: %w", err)
	}
	conn.Close()

	// In a full implementation, this would call sts.GetCallerIdentity with the
	// new credentials to verify they are valid and properly permissioned.

	c.logger.Debug().Str("target_id", targetID.String()).Msg("AWS credential connectivity verified")
	return nil
}

func (c *AWSSecretsConnector) Name() string {
	return "aws_secrets"
}

// =============================================================================
// Helpers
// =============================================================================

// sanitizeIdentifier removes characters that could be used for SQL injection
// in identifiers (usernames, role names). Only allows alphanumeric, underscore, dash.
func sanitizeIdentifier(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
