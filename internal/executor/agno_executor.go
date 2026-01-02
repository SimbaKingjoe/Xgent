package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xcode-ai/xgent-go/internal/crd"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"go.uber.org/zap"
)

// AgnoExecutor executes tasks using Agno Python SDK via bridge script
type AgnoExecutor struct {
	storage *storage.Storage
	logger  *zap.Logger
}

// NewAgnoExecutor creates a new agno executor
func NewAgnoExecutor(storage *storage.Storage, logger *zap.Logger) *AgnoExecutor {
	return &AgnoExecutor{
		storage: storage,
		logger:  logger,
	}
}

// Execute executes a task
func (e *AgnoExecutor) Execute(ctx context.Context, task *models.Task, callback ProgressCallback) error {
	e.logger.Info("Executing task",
		zap.Uint("task_id", task.ID),
		zap.String("resource_type", task.ResourceType),
		zap.String("resource_name", task.ResourceName),
	)

	// Update task status to running
	task.Status = models.TaskStatusRunning
	now := time.Now()
	task.StartedAt = &now
	if err := e.storage.Tasks().Update(task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Report progress
	if callback != nil {
		callback(task.ID, 10, models.TaskStatusRunning, "Task started", nil)
	}

	// Load resource based on type
	var result string
	var eventLogs string
	var err error

	switch task.ResourceType {
	case "bot", "robot":
		result, eventLogs, err = e.executeBot(ctx, task, callback)
	case "team":
		result, eventLogs, err = e.executeTeam(ctx, task, callback)
	default:
		err = fmt.Errorf("unsupported resource type: %s", task.ResourceType)
	}

	// Update task with result
	completed := time.Now()
	task.CompletedAt = &completed
	task.Duration = completed.Sub(*task.StartedAt).Milliseconds()

	if err != nil {
		task.Status = models.TaskStatusFailed
		task.Error = err.Error()
		e.storage.Tasks().Update(task)

		if callback != nil {
			callback(task.ID, 0, models.TaskStatusFailed, err.Error(), nil)
		}
		return err
	}

	task.Status = models.TaskStatusCompleted
	task.Result = result
	task.Progress = 100
	task.EventLogs = eventLogs
	e.storage.Tasks().Update(task)

	if callback != nil {
		callback(task.ID, 100, models.TaskStatusCompleted, "Task completed", map[string]interface{}{
			"result": result,
		})
	}

	return nil
}

// AgnoConfig represents the JSON configuration sent to the Python script
type AgnoConfig struct {
	Type      string            `json:"type"` // "robot" or "team"
	Prompt    string            `json:"prompt"`
	Model     AgnoModelConfig   `json:"model"`
	Soul      AgnoSoulConfig    `json:"soul"`
	Team      *AgnoTeamConfig   `json:"team,omitempty"`
	Context   AgnoContextConfig `json:"context"`
	SessionID string            `json:"session_id"`
	MCPTools  []AgnoMCPConfig   `json:"mcp_tools,omitempty"`

	// Execution options
	Stream       bool `json:"stream"`        // Enable streaming mode (default true)
	Debug        bool `json:"debug"`         // Enable debug mode
	DebugLevel   int  `json:"debug_level"`   // Debug level (1-3)
	ReuseSession bool `json:"reuse_session"` // Enable session reuse caching
}

type AgnoModelConfig struct {
	Provider string `json:"provider"`
	ModelID  string `json:"model_id"`
	APIKey   string `json:"api_key,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
}

type AgnoSoulConfig struct {
	Name        string `json:"name"`
	Personality string `json:"personality"`
}

// AgnoContextConfig represents execution context
type AgnoContextConfig struct {
	Cwd         string `json:"cwd,omitempty"`
	GitURL      string `json:"git_url,omitempty"`
	Branch      string `json:"branch,omitempty"`
	ProjectPath string `json:"project_path,omitempty"`
}

// AgnoMCPConfig represents MCP tool configuration
type AgnoMCPConfig struct {
	Name           string            `json:"name"`
	Type           string            `json:"type"` // stdio, sse, streamable-http
	Command        string            `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	URL            string            `json:"url,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
	SSEReadTimeout int               `json:"sse_read_timeout,omitempty"`
}

// AgnoTeamConfig represents team configuration for Python script
type AgnoTeamConfig struct {
	Name        string             `json:"name"`
	Mode        string             `json:"mode"` // coordinate, collaborate, route
	Leader      *AgnoMemberConfig  `json:"leader,omitempty"`
	Members     []AgnoMemberConfig `json:"members"`
	Description string             `json:"description,omitempty"`
}

// AgnoMemberConfig represents a team member (bot) configuration
type AgnoMemberConfig struct {
	Name        string          `json:"name"`
	Model       AgnoModelConfig `json:"model"`
	Personality string          `json:"personality"`
	Description string          `json:"description,omitempty"`
}

// executeBot executes a robot task
func (e *AgnoExecutor) executeBot(ctx context.Context, task *models.Task, callback ProgressCallback) (string, string, error) {
	// Load robot resource
	robotResource, err := e.storage.Resources().GetByName(task.WorkspaceID, task.ResourceName, models.ResourceTypeRobot)
	if err != nil {
		return "", "", fmt.Errorf("failed to load robot: %w", err)
	}

	// Parse robot spec
	parser := crd.NewParser()
	resource, err := parser.Parse([]byte(robotResource.Spec))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse robot spec: %w", err)
	}

	robot, ok := resource.(*crd.Robot)
	if !ok {
		return "", "", fmt.Errorf("invalid robot resource")
	}

	// Load soul
	soulResource, err := e.storage.Resources().GetByName(task.WorkspaceID, robot.Spec.Soul, models.ResourceTypeSoul)
	if err != nil {
		return "", "", fmt.Errorf("failed to load soul: %w", err)
	}
	soulDef, _ := parser.Parse([]byte(soulResource.Spec))
	soul := soulDef.(*crd.Soul)

	// Load mind
	mindResource, err := e.storage.Resources().GetByName(task.WorkspaceID, robot.Spec.Mind, models.ResourceTypeMind)
	if err != nil {
		return "", "", fmt.Errorf("failed to load mind: %w", err)
	}
	mindDef, _ := parser.Parse([]byte(mindResource.Spec))
	mind := mindDef.(*crd.Mind)

	// Load MCP tools if craft is configured
	var mcpTools []AgnoMCPConfig
	if robot.Spec.Craft != "" {
		mcpTools = e.loadMCPTools(task.WorkspaceID, robot.Spec.Craft, parser)
	}

	// Build session ID
	sessionID := fmt.Sprintf("task-%d", task.ID)

	// Prepare Config
	config := AgnoConfig{
		Type:      "robot",
		Prompt:    task.Prompt,
		SessionID: sessionID,
		Model: AgnoModelConfig{
			Provider: mind.Spec.Provider,
			ModelID:  mind.Spec.ModelID,
			APIKey:   mind.Spec.APIKey,
			BaseURL:  mind.Spec.BaseURL,
		},
		Soul: AgnoSoulConfig{
			Name:        robot.Metadata.Name,
			Personality: soul.Spec.Personality,
		},
		Context: AgnoContextConfig{
			GitURL: task.GitURL,
			Branch: task.BranchName,
		},
		MCPTools: mcpTools,
		// Execution options
		Stream:       true, // Default to streaming
		Debug:        false,
		DebugLevel:   2,
		ReuseSession: true,
	}

	// Execute Python script
	return e.runAgnoScript(ctx, config, task.ID, callback)
}

// executeTeam executes a team task
func (e *AgnoExecutor) executeTeam(ctx context.Context, task *models.Task, callback ProgressCallback) (string, string, error) {
	// Load team resource
	teamResource, err := e.storage.Resources().GetByName(task.WorkspaceID, task.ResourceName, models.ResourceTypeTeam)
	if err != nil {
		return "", "", fmt.Errorf("failed to load team: %w", err)
	}

	// Parse team spec
	parser := crd.NewParser()
	resource, err := parser.Parse([]byte(teamResource.Spec))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse team spec: %w", err)
	}

	teamDef, ok := resource.(*crd.Team)
	if !ok {
		return "", "", fmt.Errorf("invalid team resource")
	}

	// Build team config
	teamConfig := AgnoTeamConfig{
		Name:        teamDef.Metadata.Name,
		Mode:        string(teamDef.Spec.Mode),
		Description: teamDef.Spec.Description,
		Members:     make([]AgnoMemberConfig, 0),
	}

	// Load leader if specified
	var leaderMind *crd.Mind
	if teamDef.Spec.Leader != "" {
		leaderMember, leaderMindDef, err := e.loadRobotAsMember(task.WorkspaceID, teamDef.Spec.Leader, parser)
		if err != nil {
			return "", "", fmt.Errorf("failed to load leader robot: %w", err)
		}
		teamConfig.Leader = leaderMember
		leaderMind = leaderMindDef
	}

	// Load member robots
	for _, memberName := range teamDef.Spec.Members {
		member, mindDef, err := e.loadRobotAsMember(task.WorkspaceID, memberName, parser)
		if err != nil {
			e.logger.Warn("Failed to load member robot, skipping",
				zap.String("member", memberName),
				zap.Error(err))
			continue
		}
		teamConfig.Members = append(teamConfig.Members, *member)
		// Use first member's mind if no leader
		if leaderMind == nil {
			leaderMind = mindDef
		}
	}

	if len(teamConfig.Members) == 0 && teamConfig.Leader == nil {
		return "", "", fmt.Errorf("team has no valid members or leader")
	}

	// Use leader's mind or first member's mind for the team
	if leaderMind == nil {
		return "", "", fmt.Errorf("no mind found for team")
	}

	// Build session ID
	sessionID := fmt.Sprintf("task-%d", task.ID)

	// Prepare Config
	config := AgnoConfig{
		Type:      "team",
		Prompt:    task.Prompt,
		SessionID: sessionID,
		Model: AgnoModelConfig{
			Provider: leaderMind.Spec.Provider,
			ModelID:  leaderMind.Spec.ModelID,
			APIKey:   leaderMind.Spec.APIKey,
			BaseURL:  leaderMind.Spec.BaseURL,
		},
		Team: &teamConfig,
		Context: AgnoContextConfig{
			GitURL: task.GitURL,
			Branch: task.BranchName,
		},
		// Execution options
		Stream:       true, // Default to streaming
		Debug:        false,
		DebugLevel:   2,
		ReuseSession: true,
	}

	// Execute Python script
	return e.runAgnoScript(ctx, config, task.ID, callback)
}

// loadMCPTools loads MCP tools from a Craft resource
func (e *AgnoExecutor) loadMCPTools(workspaceID uint, craftName string, parser *crd.Parser) []AgnoMCPConfig {
	var mcpTools []AgnoMCPConfig

	// Load craft resource
	craftResource, err := e.storage.Resources().GetByName(workspaceID, craftName, models.ResourceTypeCraft)
	if err != nil {
		e.logger.Warn("Failed to load craft resource", zap.String("craft", craftName), zap.Error(err))
		return mcpTools
	}

	// Parse craft spec
	resource, err := parser.Parse([]byte(craftResource.Spec))
	if err != nil {
		e.logger.Warn("Failed to parse craft spec", zap.Error(err))
		return mcpTools
	}

	craft, ok := resource.(*crd.Craft)
	if !ok {
		e.logger.Warn("Invalid craft resource")
		return mcpTools
	}

	// Extract MCP servers
	if craft.Spec.MCP != nil {
		for _, server := range craft.Spec.MCP.Servers {
			mcpTool := AgnoMCPConfig{
				Name:    server.Name,
				Type:    "stdio", // Default to stdio for command-based servers
				Command: server.Command,
				Args:    server.Args,
				Env:     server.Env,
				Timeout: 300, // Default 5 minutes
			}
			mcpTools = append(mcpTools, mcpTool)
		}
	}

	e.logger.Info("Loaded MCP tools", zap.Int("count", len(mcpTools)))
	return mcpTools
}

// loadRobotAsMember loads a robot and returns its member config
func (e *AgnoExecutor) loadRobotAsMember(workspaceID uint, robotName string, parser *crd.Parser) (*AgnoMemberConfig, *crd.Mind, error) {
	// Load robot resource
	robotResource, err := e.storage.Resources().GetByName(workspaceID, robotName, models.ResourceTypeRobot)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load robot: %w", err)
	}

	// Parse robot spec
	resource, err := parser.Parse([]byte(robotResource.Spec))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse robot spec: %w", err)
	}

	robot, ok := resource.(*crd.Robot)
	if !ok {
		return nil, nil, fmt.Errorf("invalid robot resource")
	}

	// Load soul
	soulResource, err := e.storage.Resources().GetByName(workspaceID, robot.Spec.Soul, models.ResourceTypeSoul)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load soul: %w", err)
	}
	soulDef, _ := parser.Parse([]byte(soulResource.Spec))
	soul := soulDef.(*crd.Soul)

	// Load mind
	mindResource, err := e.storage.Resources().GetByName(workspaceID, robot.Spec.Mind, models.ResourceTypeMind)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load mind: %w", err)
	}
	mindDef, _ := parser.Parse([]byte(mindResource.Spec))
	mind := mindDef.(*crd.Mind)

	member := &AgnoMemberConfig{
		Name:        robot.Metadata.Name,
		Personality: soul.Spec.Personality,
		Model: AgnoModelConfig{
			Provider: mind.Spec.Provider,
			ModelID:  mind.Spec.ModelID,
			APIKey:   mind.Spec.APIKey,
			BaseURL:  mind.Spec.BaseURL,
		},
	}

	return member, mind, nil
}

// runAgnoScript runs the Python bridge script
func (e *AgnoExecutor) runAgnoScript(ctx context.Context, config AgnoConfig, taskID uint, callback ProgressCallback) (string, string, error) {
	// Locate script
	// Assuming running from project root
	cwd, _ := os.Getwd()
	scriptPath := filepath.Join(cwd, "scripts", "agno_runner.py")

	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("agno runner script not found at %s", scriptPath)
	}

	// Prepare command
	cmd := exec.CommandContext(ctx, "python3", scriptPath)
	// cmd := exec.CommandContext(ctx, "python", scriptPath) // Try python if python3 fails?

	// Set proxy environment variables
	cmd.Env = append(os.Environ(),
		"https_proxy=http://127.0.0.1:7890",
		"http_proxy=http://127.0.0.1:7890",
		"all_proxy=socks5://127.0.0.1:7890",
		"HTTPS_PROXY=http://127.0.0.1:7890",
		"HTTP_PROXY=http://127.0.0.1:7890",
		"ALL_PROXY=socks5://127.0.0.1:7890",
	)

	// Stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Stderr pipe
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return "", "", fmt.Errorf("failed to start python script: %w", err)
	}

	// Write config to stdin
	go func() {
		defer stdin.Close()
		json.NewEncoder(stdin).Encode(config)
	}()

	// Read output
	var fullContent strings.Builder
	scanner := bufio.NewScanner(stdout)

	go func() {
		// Read stderr for debugging
		stderrScanner := bufio.NewScanner(stderr)
		for stderrScanner.Scan() {
			e.logger.Error("Python script stderr", zap.String("line", stderrScanner.Text()))
		}
	}()

	var lastError string
	var eventLogs []string
	for scanner.Scan() {
		line := scanner.Text()
		e.logger.Debug("Python script output", zap.String("line", line))

		var event struct {
			Type    string                 `json:"type"`
			Content string                 `json:"content"`
			Details map[string]interface{} `json:"details,omitempty"`
		}

		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Not JSON, treat as raw log or ignored
			continue
		}

		// Store event for logs
		eventLogs = append(eventLogs, line)

		// Process specific event types (callback is called within each case)
		switch event.Type {
		case "started":
			e.logger.Info("Agno execution started", zap.String("content", event.Content))
			if callback != nil {
				callback(taskID, 20, models.TaskStatusRunning, "Agent started", map[string]interface{}{
					"type": "started",
				})
			}

		case "content":
			fullContent.WriteString(event.Content)
			if callback != nil {
				callback(taskID, 50, models.TaskStatusRunning, event.Content, map[string]interface{}{
					"type": "content",
				})
			}

		case "run_started", "team_run_started":
			e.logger.Info("Agent/Team run started", zap.String("content", event.Content))
			if callback != nil {
				callback(taskID, 30, models.TaskStatusRunning, event.Content, map[string]interface{}{
					"type": event.Type,
				})
			}

		case "run_completed", "team_run_completed":
			e.logger.Info("Agent/Team run completed", zap.String("content", event.Content))
			if callback != nil {
				callback(taskID, 90, models.TaskStatusRunning, event.Content, map[string]interface{}{
					"type": event.Type,
				})
			}

		case "tool_call_started", "member_tool_started":
			e.logger.Info("Tool call started", zap.Any("details", event.Details))
			if callback != nil {
				callback(taskID, 60, models.TaskStatusRunning, "Tool call started", map[string]interface{}{
					"type":    event.Type,
					"details": event.Details,
				})
			}

		case "tool_call_completed", "member_tool_completed":
			e.logger.Info("Tool call completed", zap.Any("details", event.Details))
			if callback != nil {
				callback(taskID, 70, models.TaskStatusRunning, "Tool call completed", map[string]interface{}{
					"type":    event.Type,
					"details": event.Details,
				})
			}

		case "reasoning":
			e.logger.Info("Team reasoning step", zap.Any("details", event.Details))
			if callback != nil {
				callback(taskID, 55, models.TaskStatusRunning, "Reasoning", map[string]interface{}{
					"type":    "reasoning",
					"details": event.Details,
				})
			}

		case "mcp_connected":
			e.logger.Info("MCP tool connected", zap.String("content", event.Content))
			if callback != nil {
				callback(taskID, 25, models.TaskStatusRunning, event.Content, map[string]interface{}{
					"type": "mcp_connected",
				})
			}

		case "thinking_step":
			e.logger.Info("Thinking step", zap.String("content", event.Content), zap.Any("details", event.Details))
			if callback != nil {
				callback(taskID, 40, models.TaskStatusRunning, event.Content, map[string]interface{}{
					"type":    "thinking_step",
					"details": event.Details,
				})
			}

		case "session_reused":
			e.logger.Info("Session reused", zap.String("content", event.Content))
			if callback != nil {
				callback(taskID, 25, models.TaskStatusRunning, event.Content, map[string]interface{}{
					"type": "session_reused",
				})
			}

		case "git_downloaded":
			e.logger.Info("Git code downloaded", zap.String("content", event.Content), zap.Any("details", event.Details))
			if callback != nil {
				callback(taskID, 15, models.TaskStatusRunning, event.Content, map[string]interface{}{
					"type":    "git_downloaded",
					"details": event.Details,
				})
			}

		case "warning":
			e.logger.Warn("Agno script warning", zap.String("content", event.Content))

		case "completed":
			e.logger.Info("Agno execution completed")
			// Content is already in fullContent from "content" events

		case "cancelled":
			e.logger.Info("Agno execution cancelled", zap.String("content", event.Content))
			return fullContent.String(), strings.Join(eventLogs, "\n"), nil

		case "error":
			lastError = event.Content
			e.logger.Error("Agno script reported error", zap.String("error", event.Content))
		}
	}

	if err := cmd.Wait(); err != nil {
		if lastError != "" {
			return "", "", fmt.Errorf("python script error: %s", lastError)
		}
		return "", "", fmt.Errorf("python script finished with error: %w", err)
	}

	return fullContent.String(), strings.Join(eventLogs, "\n"), nil
}
