package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/internal/ssh"
	"github.com/bison/api-server/pkg/logger"
)

// Ensure metav1 is used
var _ = metav1.Now

const (
	OnboardingJobsConfigMap = "bison-onboarding-jobs"
)

// OnboardingJobStatus represents the status of an onboarding job
type OnboardingJobStatus string

const (
	JobStatusPending   OnboardingJobStatus = "pending"
	JobStatusRunning   OnboardingJobStatus = "running"
	JobStatusSuccess   OnboardingJobStatus = "success"
	JobStatusFailed    OnboardingJobStatus = "failed"
	JobStatusCancelled OnboardingJobStatus = "cancelled"
)

// SubStepStatus represents the status of a sub-step
type SubStepStatus string

const (
	SubStepPending SubStepStatus = "pending"
	SubStepRunning SubStepStatus = "running"
	SubStepSuccess SubStepStatus = "success"
	SubStepFailed  SubStepStatus = "failed"
	SubStepSkipped SubStepStatus = "skipped"
)

// SubStep represents a sub-step within a main step
type SubStep struct {
	Name   string        `json:"name"`
	Status SubStepStatus `json:"status"`
	Error  string        `json:"error,omitempty"`
}

// OnboardingJob represents a node onboarding job
type OnboardingJob struct {
	ID           string              `json:"id"`
	NodeIP       string              `json:"nodeIP"`
	NodeName     string              `json:"nodeName,omitempty"`
	Platform     NodePlatform        `json:"platform"`
	Status       OnboardingJobStatus `json:"status"`
	CurrentStep  int                 `json:"currentStep"`
	TotalSteps   int                 `json:"totalSteps"`
	StepMessage  string              `json:"stepMessage"`
	SubSteps     []SubStep           `json:"subSteps,omitempty"`
	ErrorMessage string              `json:"errorMessage,omitempty"`
	CreatedAt    time.Time           `json:"createdAt"`
	UpdatedAt    time.Time           `json:"updatedAt"`
	CompletedAt  *time.Time          `json:"completedAt,omitempty"`
}

// OnboardingRequest represents a request to onboard a new node
type OnboardingRequest struct {
	NodeIP      string `json:"nodeIP" binding:"required"`
	SSHPort     int    `json:"sshPort"`
	SSHUsername string `json:"sshUsername" binding:"required"`
	AuthMethod  string `json:"authMethod" binding:"required,oneof=password privateKey"`
	Password    string `json:"password"`
	PrivateKey  string `json:"privateKey"`
}

// OnboardingService handles node onboarding operations
type OnboardingService struct {
	k8sClient       *k8s.Client
	nodeSvc         *NodeService
	initScriptSvc   *InitScriptService
	runningJobs     map[string]context.CancelFunc
	runningJobsMu   sync.RWMutex
}

// NewOnboardingService creates a new OnboardingService
func NewOnboardingService(k8sClient *k8s.Client, nodeSvc *NodeService, initScriptSvc *InitScriptService) *OnboardingService {
	return &OnboardingService{
		k8sClient:     k8sClient,
		nodeSvc:       nodeSvc,
		initScriptSvc: initScriptSvc,
		runningJobs:   make(map[string]context.CancelFunc),
	}
}

// StartOnboarding starts a new node onboarding job
func (s *OnboardingService) StartOnboarding(ctx context.Context, req *OnboardingRequest) (*OnboardingJob, error) {
	logger.Info("Starting node onboarding", "nodeIP", req.NodeIP)

	// Set defaults
	if req.SSHPort == 0 {
		req.SSHPort = 22
	}

	// Validate authentication
	if req.AuthMethod == "password" && req.Password == "" {
		return nil, fmt.Errorf("password is required for password authentication")
	}
	if req.AuthMethod == "privateKey" && req.PrivateKey == "" {
		return nil, fmt.Errorf("private key is required for private key authentication")
	}

	// Check if there's already a running job for this IP
	jobs, err := s.ListJobs(ctx)
	if err != nil {
		return nil, err
	}
	for _, job := range jobs {
		if job.NodeIP == req.NodeIP && (job.Status == JobStatusPending || job.Status == JobStatusRunning) {
			return nil, fmt.Errorf("there is already a running onboarding job for this IP: %s", job.ID)
		}
	}

	// Create job
	job := &OnboardingJob{
		ID:          fmt.Sprintf("job-%d", time.Now().UnixNano()),
		NodeIP:      req.NodeIP,
		Status:      JobStatusPending,
		CurrentStep: 0,
		TotalSteps:  9,
		StepMessage: "Job created, waiting to start",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save job
	if err := s.saveJob(ctx, job); err != nil {
		return nil, err
	}

	// Start async execution
	jobCtx, cancel := context.WithCancel(context.Background())
	s.runningJobsMu.Lock()
	s.runningJobs[job.ID] = cancel
	s.runningJobsMu.Unlock()

	go s.executeOnboarding(jobCtx, job, req)

	return job, nil
}

// GetJob returns a specific job by ID
func (s *OnboardingService) GetJob(ctx context.Context, jobID string) (*OnboardingJob, error) {
	jobs, err := s.getJobsMap(ctx)
	if err != nil {
		return nil, err
	}

	jobData, ok := jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	var job OnboardingJob
	if err := json.Unmarshal([]byte(jobData), &job); err != nil {
		return nil, fmt.Errorf("failed to parse job data: %w", err)
	}

	return &job, nil
}

// ListJobs returns all onboarding jobs
func (s *OnboardingService) ListJobs(ctx context.Context) ([]*OnboardingJob, error) {
	jobs, err := s.getJobsMap(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*OnboardingJob, 0, len(jobs))
	for _, jobData := range jobs {
		var job OnboardingJob
		if err := json.Unmarshal([]byte(jobData), &job); err != nil {
			continue
		}
		result = append(result, &job)
	}

	return result, nil
}

// CancelJob cancels a running job
func (s *OnboardingService) CancelJob(ctx context.Context, jobID string) error {
	logger.Info("Cancelling onboarding job", "jobID", jobID)

	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return err
	}

	if job.Status != JobStatusPending && job.Status != JobStatusRunning {
		return fmt.Errorf("job is not running: %s", job.Status)
	}

	// Cancel the job context
	s.runningJobsMu.Lock()
	if cancel, ok := s.runningJobs[jobID]; ok {
		cancel()
		delete(s.runningJobs, jobID)
	}
	s.runningJobsMu.Unlock()

	// Update job status
	job.Status = JobStatusCancelled
	job.StepMessage = "Job cancelled by user"
	job.UpdatedAt = time.Now()
	now := time.Now()
	job.CompletedAt = &now

	return s.saveJob(ctx, job)
}

// executeOnboarding executes the onboarding process
func (s *OnboardingService) executeOnboarding(ctx context.Context, job *OnboardingJob, req *OnboardingRequest) {
	defer func() {
		s.runningJobsMu.Lock()
		delete(s.runningJobs, job.ID)
		s.runningJobsMu.Unlock()
	}()

	// Update job status to running
	job.Status = JobStatusRunning
	job.UpdatedAt = time.Now()
	s.saveJob(context.Background(), job)

	// Create SSH executor for target node
	sshConfig := &ssh.Config{
		Host:       req.NodeIP,
		Port:       req.SSHPort,
		Username:   req.SSHUsername,
		AuthMethod: ssh.AuthMethod(req.AuthMethod),
		Password:   req.Password,
		PrivateKey: req.PrivateKey,
		Timeout:    30 * time.Second,
	}
	executor := ssh.NewExecutor(sshConfig)
	defer executor.Close()

	// Step 1: Connection test
	if err := s.stepConnectionTest(ctx, job, executor); err != nil {
		s.failJob(job, err)
		return
	}

	// Step 2: Platform detection
	if err := s.stepPlatformDetection(ctx, job, executor); err != nil {
		s.failJob(job, err)
		return
	}

	// Step 3: Environment check
	if err := s.stepEnvironmentCheck(ctx, job, executor); err != nil {
		s.failJob(job, err)
		return
	}

	// Step 4: Pre-join scripts
	if err := s.stepPreJoinScripts(ctx, job, executor); err != nil {
		s.failJob(job, err)
		return
	}

	// Step 5: Get join token
	joinCommand, err := s.stepGetJoinToken(ctx, job)
	if err != nil {
		s.failJob(job, err)
		return
	}

	// Step 6: Execute kubeadm join
	if err := s.stepKubeadmJoin(ctx, job, executor, joinCommand); err != nil {
		s.failJob(job, err)
		return
	}

	// Step 7: Post-join scripts
	if err := s.stepPostJoinScripts(ctx, job, executor); err != nil {
		s.failJob(job, err)
		return
	}

	// Step 8: Wait for node ready
	if err := s.stepWaitForNodeReady(ctx, job); err != nil {
		s.failJob(job, err)
		return
	}

	// Step 9: Enable node
	if err := s.stepEnableNode(ctx, job); err != nil {
		s.failJob(job, err)
		return
	}

	// Mark job as successful
	job.Status = JobStatusSuccess
	job.StepMessage = "Node onboarding completed successfully"
	job.UpdatedAt = time.Now()
	now := time.Now()
	job.CompletedAt = &now
	s.saveJob(context.Background(), job)

	logger.Info("Node onboarding completed successfully", "nodeIP", job.NodeIP, "nodeName", job.NodeName)
}

func (s *OnboardingService) stepConnectionTest(ctx context.Context, job *OnboardingJob, executor *ssh.Executor) error {
	job.CurrentStep = 1
	job.StepMessage = "Testing SSH connection..."
	job.UpdatedAt = time.Now()
	s.saveJob(context.Background(), job)

	if err := executor.TestConnection(ctx); err != nil {
		return fmt.Errorf("SSH connection test failed: %w", err)
	}

	return nil
}

func (s *OnboardingService) stepPlatformDetection(ctx context.Context, job *OnboardingJob, executor *ssh.Executor) error {
	job.CurrentStep = 2
	job.StepMessage = "Detecting node platform..."
	job.UpdatedAt = time.Now()
	s.saveJob(context.Background(), job)

	info, err := executor.GetHostInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect platform: %w", err)
	}

	job.Platform = NodePlatform{
		OS:      info["os"],
		Version: info["version"],
		Arch:    info["arch"],
	}

	if info["hostname"] != "" {
		job.NodeName = info["hostname"]
	}

	job.StepMessage = fmt.Sprintf("Detected: %s %s (%s)", job.Platform.OS, job.Platform.Version, job.Platform.Arch)
	job.UpdatedAt = time.Now()
	s.saveJob(context.Background(), job)

	return nil
}

func (s *OnboardingService) stepEnvironmentCheck(ctx context.Context, job *OnboardingJob, executor *ssh.Executor) error {
	job.CurrentStep = 3
	job.StepMessage = "Checking environment..."
	job.UpdatedAt = time.Now()
	s.saveJob(context.Background(), job)

	// Check if kubeadm is installed
	if !executor.CheckCommand(ctx, "kubeadm") {
		return fmt.Errorf("kubeadm is not installed on the target node")
	}

	// Check if kubelet is installed
	if !executor.CheckCommand(ctx, "kubelet") {
		return fmt.Errorf("kubelet is not installed on the target node")
	}

	return nil
}

func (s *OnboardingService) stepPreJoinScripts(ctx context.Context, job *OnboardingJob, executor *ssh.Executor) error {
	job.CurrentStep = 4
	job.StepMessage = "Executing pre-join scripts..."
	job.UpdatedAt = time.Now()
	s.saveJob(context.Background(), job)

	// Get init scripts for pre-join phase
	scripts, err := s.initScriptSvc.GetScriptsForPhase(ctx, PhasePreJoin, job.Platform)
	if err != nil {
		return fmt.Errorf("failed to get pre-join scripts: %w", err)
	}

	if len(scripts) == 0 {
		job.StepMessage = "No pre-join scripts to execute"
		job.UpdatedAt = time.Now()
		s.saveJob(context.Background(), job)
		return nil
	}

	// Initialize sub-steps
	job.SubSteps = make([]SubStep, len(scripts))
	for i, script := range scripts {
		job.SubSteps[i] = SubStep{
			Name:   script.Group.Name,
			Status: SubStepPending,
		}
	}
	s.saveJob(context.Background(), job)

	// Get variables for script replacement
	cpConfig, _ := s.initScriptSvc.GetControlPlaneConfig(ctx)
	controlPlaneIP := ""
	if cpConfig != nil {
		controlPlaneIP = cpConfig.Host
	}
	vars := map[string]string{
		"NODE_IP":          job.NodeIP,
		"NODE_NAME":        job.NodeName,
		"CONTROL_PLANE_IP": controlPlaneIP,
	}

	// Execute scripts
	for stepIdx, script := range scripts {
		job.SubSteps[stepIdx].Status = SubStepRunning
		job.StepMessage = fmt.Sprintf("Executing: %s", script.Group.Name)
		job.UpdatedAt = time.Now()
		s.saveJob(context.Background(), job)

		// Replace variables in script content
		content := ReplaceVariables(script.Script.Content, vars)

		// Execute script
		result := executor.ExecuteScript(ctx, content)
		if result.Error != nil || result.ExitCode != 0 {
			job.SubSteps[stepIdx].Status = SubStepFailed
			errMsg := result.Stderr
			if result.Error != nil {
				errMsg = result.Error.Error()
			}
			job.SubSteps[stepIdx].Error = errMsg
			s.saveJob(context.Background(), job)
			return fmt.Errorf("pre-join script '%s' failed: %s", script.Group.Name, errMsg)
		}

		job.SubSteps[stepIdx].Status = SubStepSuccess
		s.saveJob(context.Background(), job)
	}

	job.SubSteps = nil // Clear sub-steps after completion
	return nil
}

func (s *OnboardingService) stepGetJoinToken(ctx context.Context, job *OnboardingJob) (string, error) {
	job.CurrentStep = 5
	job.StepMessage = "Getting join token from control plane..."
	job.UpdatedAt = time.Now()
	s.saveJob(context.Background(), job)

	// Get control plane config
	cpConfig, err := s.initScriptSvc.GetControlPlaneConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get control plane config: %w", err)
	}

	if cpConfig.Host == "" {
		return "", fmt.Errorf("control plane host is not configured")
	}

	// Create SSH executor for control plane
	cpSSHConfig := &ssh.Config{
		Host:       cpConfig.Host,
		Port:       cpConfig.SSHPort,
		Username:   cpConfig.SSHUser,
		AuthMethod: ssh.AuthMethod(cpConfig.AuthMethod),
		Password:   cpConfig.Password,
		PrivateKey: cpConfig.PrivateKey,
		Timeout:    30 * time.Second,
	}
	cpExecutor := ssh.NewExecutor(cpSSHConfig)
	defer cpExecutor.Close()

	if err := cpExecutor.Connect(ctx); err != nil {
		return "", fmt.Errorf("failed to connect to control plane: %w", err)
	}

	// Generate join command
	result := cpExecutor.Execute(ctx, "kubeadm token create --print-join-command")
	if result.Error != nil || result.ExitCode != 0 {
		errMsg := result.Stderr
		if result.Error != nil {
			errMsg = result.Error.Error()
		}
		return "", fmt.Errorf("failed to generate join command: %s", errMsg)
	}

	joinCommand := result.Stdout
	if joinCommand == "" {
		return "", fmt.Errorf("empty join command returned")
	}

	return joinCommand, nil
}

func (s *OnboardingService) stepKubeadmJoin(ctx context.Context, job *OnboardingJob, executor *ssh.Executor, joinCommand string) error {
	job.CurrentStep = 6
	job.StepMessage = "Executing kubeadm join..."
	job.UpdatedAt = time.Now()
	s.saveJob(context.Background(), job)

	// Execute kubeadm join with a longer timeout
	joinCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	result := executor.Execute(joinCtx, joinCommand)
	if result.Error != nil || result.ExitCode != 0 {
		errMsg := result.Stderr
		if result.Error != nil {
			errMsg = result.Error.Error()
		}
		return fmt.Errorf("kubeadm join failed: %s", errMsg)
	}

	return nil
}

func (s *OnboardingService) stepPostJoinScripts(ctx context.Context, job *OnboardingJob, executor *ssh.Executor) error {
	job.CurrentStep = 7
	job.StepMessage = "Executing post-join scripts..."
	job.UpdatedAt = time.Now()
	s.saveJob(context.Background(), job)

	// Get init scripts for post-join phase
	scripts, err := s.initScriptSvc.GetScriptsForPhase(ctx, PhasePostJoin, job.Platform)
	if err != nil {
		return fmt.Errorf("failed to get post-join scripts: %w", err)
	}

	if len(scripts) == 0 {
		job.StepMessage = "No post-join scripts to execute"
		job.UpdatedAt = time.Now()
		s.saveJob(context.Background(), job)
		return nil
	}

	// Initialize sub-steps
	job.SubSteps = make([]SubStep, len(scripts))
	for i, script := range scripts {
		job.SubSteps[i] = SubStep{
			Name:   script.Group.Name,
			Status: SubStepPending,
		}
	}
	s.saveJob(context.Background(), job)

	// Get variables for script replacement
	cpConfig, _ := s.initScriptSvc.GetControlPlaneConfig(ctx)
	controlPlaneIP := ""
	if cpConfig != nil {
		controlPlaneIP = cpConfig.Host
	}
	vars := map[string]string{
		"NODE_IP":          job.NodeIP,
		"NODE_NAME":        job.NodeName,
		"CONTROL_PLANE_IP": controlPlaneIP,
	}

	// Execute scripts
	for stepIdx, script := range scripts {
		job.SubSteps[stepIdx].Status = SubStepRunning
		job.StepMessage = fmt.Sprintf("Executing: %s", script.Group.Name)
		job.UpdatedAt = time.Now()
		s.saveJob(context.Background(), job)

		// Replace variables in script content
		content := ReplaceVariables(script.Script.Content, vars)

		// Execute script
		result := executor.ExecuteScript(ctx, content)
		if result.Error != nil || result.ExitCode != 0 {
			job.SubSteps[stepIdx].Status = SubStepFailed
			errMsg := result.Stderr
			if result.Error != nil {
				errMsg = result.Error.Error()
			}
			job.SubSteps[stepIdx].Error = errMsg
			s.saveJob(context.Background(), job)
			return fmt.Errorf("post-join script '%s' failed: %s", script.Group.Name, errMsg)
		}

		job.SubSteps[stepIdx].Status = SubStepSuccess
		s.saveJob(context.Background(), job)
	}

	job.SubSteps = nil // Clear sub-steps after completion
	return nil
}

func (s *OnboardingService) stepWaitForNodeReady(ctx context.Context, job *OnboardingJob) error {
	job.CurrentStep = 8
	job.StepMessage = "Waiting for node to be ready..."
	job.UpdatedAt = time.Now()
	s.saveJob(context.Background(), job)

	// Wait for node to appear and become ready
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for node to be ready")
		case <-ticker.C:
			// Try to find the node
			nodes, err := s.k8sClient.ListNodes(ctx)
			if err != nil {
				continue
			}

			for _, node := range nodes.Items {
				// Match by IP or hostname
				nodeIP := ""
				for _, addr := range node.Status.Addresses {
					if addr.Type == corev1.NodeInternalIP {
						nodeIP = addr.Address
						break
					}
				}

				if nodeIP == job.NodeIP || node.Name == job.NodeName {
					job.NodeName = node.Name

					// Check if node is ready
					for _, cond := range node.Status.Conditions {
						if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
							job.StepMessage = fmt.Sprintf("Node %s is ready", node.Name)
							job.UpdatedAt = time.Now()
							s.saveJob(context.Background(), job)
							return nil
						}
					}
				}
			}
		}
	}
}

func (s *OnboardingService) stepEnableNode(ctx context.Context, job *OnboardingJob) error {
	job.CurrentStep = 9
	job.StepMessage = "Enabling node in Bison..."
	job.UpdatedAt = time.Now()
	s.saveJob(context.Background(), job)

	if job.NodeName == "" {
		return fmt.Errorf("node name is not set")
	}

	// Enable node in Bison (add to shared pool)
	if err := s.nodeSvc.EnableNode(ctx, job.NodeName); err != nil {
		return fmt.Errorf("failed to enable node: %w", err)
	}

	return nil
}

func (s *OnboardingService) failJob(job *OnboardingJob, err error) {
	job.Status = JobStatusFailed
	job.ErrorMessage = err.Error()
	job.UpdatedAt = time.Now()
	now := time.Now()
	job.CompletedAt = &now
	s.saveJob(context.Background(), job)

	logger.Error("Node onboarding failed", "nodeIP", job.NodeIP, "error", err)
}

func (s *OnboardingService) saveJob(ctx context.Context, job *OnboardingJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, OnboardingJobsConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new ConfigMap
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      OnboardingJobsConfigMap,
					Namespace: BisonNamespace,
				},
				Data: map[string]string{
					job.ID: string(data),
				},
			}
			return s.k8sClient.CreateConfigMap(ctx, BisonNamespace, cm)
		}
		return fmt.Errorf("failed to get jobs config: %w", err)
	}

	// Update existing ConfigMap
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[job.ID] = string(data)

	return s.k8sClient.UpdateConfigMap(ctx, BisonNamespace, cm)
}

func (s *OnboardingService) getJobsMap(ctx context.Context) (map[string]string, error) {
	cm, err := s.k8sClient.GetConfigMap(ctx, BisonNamespace, OnboardingJobsConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to get jobs config: %w", err)
	}

	if cm.Data == nil {
		return make(map[string]string), nil
	}

	return cm.Data, nil
}

// TestControlPlaneConnection tests the SSH connection to the control plane
func (s *OnboardingService) TestControlPlaneConnection(ctx context.Context) error {
	cpConfig, err := s.initScriptSvc.GetControlPlaneConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get control plane config: %w", err)
	}

	if cpConfig.Host == "" {
		return fmt.Errorf("control plane host is not configured")
	}

	sshConfig := &ssh.Config{
		Host:       cpConfig.Host,
		Port:       cpConfig.SSHPort,
		Username:   cpConfig.SSHUser,
		AuthMethod: ssh.AuthMethod(cpConfig.AuthMethod),
		Password:   cpConfig.Password,
		PrivateKey: cpConfig.PrivateKey,
		Timeout:    30 * time.Second,
	}

	executor := ssh.NewExecutor(sshConfig)
	defer executor.Close()

	if err := executor.TestConnection(ctx); err != nil {
		return fmt.Errorf("SSH connection test failed: %w", err)
	}

	// Also verify kubeadm is available
	if !executor.CheckCommand(ctx, "kubeadm") {
		return fmt.Errorf("kubeadm is not available on the control plane")
	}

	return nil
}
