package ssh

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// AuthMethod represents the SSH authentication method
type AuthMethod string

const (
	AuthMethodPassword   AuthMethod = "password"
	AuthMethodPrivateKey AuthMethod = "privateKey"
)

// Config holds SSH connection configuration
type Config struct {
	Host       string
	Port       int
	Username   string
	AuthMethod AuthMethod
	Password   string
	PrivateKey string // PEM encoded private key content
	Timeout    time.Duration
}

// CommandResult holds the result of a remote command execution
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
}

// Executor handles SSH connections and remote command execution
type Executor struct {
	config *Config
	client *ssh.Client
	mu     sync.Mutex
}

// NewExecutor creates a new SSH executor with the given configuration
func NewExecutor(config *Config) *Executor {
	if config.Port == 0 {
		config.Port = 22
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &Executor{
		config: config,
	}
}

// Connect establishes an SSH connection to the remote host
func (e *Executor) Connect(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.client != nil {
		return nil // Already connected
	}

	var authMethods []ssh.AuthMethod

	switch e.config.AuthMethod {
	case AuthMethodPassword:
		if e.config.Password == "" {
			return fmt.Errorf("password is required for password authentication")
		}
		authMethods = append(authMethods, ssh.Password(e.config.Password))

	case AuthMethodPrivateKey:
		if e.config.PrivateKey == "" {
			return fmt.Errorf("private key is required for private key authentication")
		}
		signer, err := ssh.ParsePrivateKey([]byte(e.config.PrivateKey))
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))

	default:
		return fmt.Errorf("unsupported authentication method: %s", e.config.AuthMethod)
	}

	sshConfig := &ssh.ClientConfig{
		User:            e.config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Consider using known_hosts in production
		Timeout:         e.config.Timeout,
	}

	addr := fmt.Sprintf("%s:%d", e.config.Host, e.config.Port)

	// Use context for connection timeout
	var client *ssh.Client
	var err error

	done := make(chan struct{})
	go func() {
		client, err = ssh.Dial("tcp", addr, sshConfig)
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %w", addr, err)
		}
	}

	e.client = client
	return nil
}

// Execute runs a command on the remote host and returns the result
func (e *Executor) Execute(ctx context.Context, command string) *CommandResult {
	e.mu.Lock()
	if e.client == nil {
		e.mu.Unlock()
		return &CommandResult{
			ExitCode: -1,
			Error:    fmt.Errorf("not connected"),
		}
	}
	client := e.client
	e.mu.Unlock()

	session, err := client.NewSession()
	if err != nil {
		return &CommandResult{
			ExitCode: -1,
			Error:    fmt.Errorf("failed to create session: %w", err),
		}
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Run command with context cancellation support
	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case <-ctx.Done():
		// Try to close the session to stop the command
		session.Close()
		return &CommandResult{
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: -1,
			Error:    ctx.Err(),
		}
	case err := <-done:
		result := &CommandResult{
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: 0,
		}

		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				result.ExitCode = exitErr.ExitStatus()
			} else {
				result.ExitCode = -1
				result.Error = err
			}
		}

		return result
	}
}

// ExecuteWithTimeout runs a command with a specific timeout
func (e *Executor) ExecuteWithTimeout(command string, timeout time.Duration) *CommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return e.Execute(ctx, command)
}

// ExecuteScript executes a shell script on the remote host
// The script content is passed via stdin to avoid escaping issues
func (e *Executor) ExecuteScript(ctx context.Context, script string) *CommandResult {
	e.mu.Lock()
	if e.client == nil {
		e.mu.Unlock()
		return &CommandResult{
			ExitCode: -1,
			Error:    fmt.Errorf("not connected"),
		}
	}
	client := e.client
	e.mu.Unlock()

	session, err := client.NewSession()
	if err != nil {
		return &CommandResult{
			ExitCode: -1,
			Error:    fmt.Errorf("failed to create session: %w", err),
		}
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Pass script via stdin
	stdin, err := session.StdinPipe()
	if err != nil {
		return &CommandResult{
			ExitCode: -1,
			Error:    fmt.Errorf("failed to create stdin pipe: %w", err),
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- session.Run("bash -s")
	}()

	// Write script to stdin
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, script)
	}()

	select {
	case <-ctx.Done():
		session.Close()
		return &CommandResult{
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: -1,
			Error:    ctx.Err(),
		}
	case err := <-done:
		result := &CommandResult{
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: 0,
		}

		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				result.ExitCode = exitErr.ExitStatus()
			} else {
				result.ExitCode = -1
				result.Error = err
			}
		}

		return result
	}
}

// TestConnection tests if the SSH connection can be established
func (e *Executor) TestConnection(ctx context.Context) error {
	if err := e.Connect(ctx); err != nil {
		return err
	}

	// Run a simple command to verify the connection works
	result := e.Execute(ctx, "echo ok")
	if result.Error != nil {
		return result.Error
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("connection test failed: %s", result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) != "ok" {
		return fmt.Errorf("unexpected response: %s", result.Stdout)
	}

	return nil
}

// Close closes the SSH connection
func (e *Executor) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.client != nil {
		err := e.client.Close()
		e.client = nil
		return err
	}
	return nil
}

// IsConnected returns true if there is an active SSH connection
func (e *Executor) IsConnected() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.client != nil
}

// GetHostInfo retrieves basic host information (OS, architecture, etc.)
func (e *Executor) GetHostInfo(ctx context.Context) (map[string]string, error) {
	info := make(map[string]string)

	// Get OS information
	osResult := e.Execute(ctx, "cat /etc/os-release 2>/dev/null | grep -E '^(ID|VERSION_ID)=' | cut -d'=' -f2 | tr -d '\"'")
	if osResult.Error == nil && osResult.ExitCode == 0 {
		lines := strings.Split(strings.TrimSpace(osResult.Stdout), "\n")
		if len(lines) >= 1 {
			info["os"] = strings.TrimSpace(lines[0])
		}
		if len(lines) >= 2 {
			info["version"] = strings.TrimSpace(lines[1])
		}
	}

	// Get architecture
	archResult := e.Execute(ctx, "uname -m")
	if archResult.Error == nil && archResult.ExitCode == 0 {
		arch := strings.TrimSpace(archResult.Stdout)
		// Normalize architecture names
		switch arch {
		case "x86_64":
			arch = "amd64"
		case "aarch64":
			arch = "arm64"
		}
		info["arch"] = arch
	}

	// Get hostname
	hostnameResult := e.Execute(ctx, "hostname")
	if hostnameResult.Error == nil && hostnameResult.ExitCode == 0 {
		info["hostname"] = strings.TrimSpace(hostnameResult.Stdout)
	}

	return info, nil
}

// CheckCommand checks if a command exists on the remote host
func (e *Executor) CheckCommand(ctx context.Context, command string) bool {
	result := e.Execute(ctx, fmt.Sprintf("command -v %s", command))
	return result.Error == nil && result.ExitCode == 0
}

// DialFunc returns a function that can be used as a proxy dialer
func (e *Executor) DialFunc() func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		e.mu.Lock()
		client := e.client
		e.mu.Unlock()

		if client == nil {
			return nil, fmt.Errorf("not connected")
		}
		return client.Dial(network, addr)
	}
}
