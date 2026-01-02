package executor

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/xcode-ai/xgent-go/internal/crd"
	"github.com/xcode-ai/xgent-go/internal/llm"
	"github.com/xcode-ai/xgent-go/internal/storage"
	"github.com/xcode-ai/xgent-go/internal/storage/models"
	"go.uber.org/zap"
)

// Executor executes tasks using LLM calls
type Executor struct {
	storage    *storage.Storage
	logger     *zap.Logger
	llmClients map[string]llm.Client
}

// New creates a new executor
func New(storage *storage.Storage, logger *zap.Logger) *Executor {
	return &Executor{
		storage:    storage,
		logger:     logger,
		llmClients: make(map[string]llm.Client),
	}
}

// ProgressCallback is an alias for models.ProgressCallback
type ProgressCallback = models.ProgressCallback

// Execute executes a task
func (e *Executor) Execute(ctx context.Context, task *models.Task, callback ProgressCallback) error {
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
	var err error

	switch task.ResourceType {
	case "bot", "robot":
		result, err = e.executeBot(ctx, task, callback)
	case "team":
		result, err = e.executeTeam(ctx, task, callback)
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
	e.storage.Tasks().Update(task)

	if callback != nil {
		callback(task.ID, 100, models.TaskStatusCompleted, "Task completed", map[string]interface{}{
			"result": result,
		})
	}

	return nil
}

// executeBot executes a robot task
func (e *Executor) executeBot(ctx context.Context, task *models.Task, callback ProgressCallback) (string, error) {
	// Load robot resource
	robotResource, err := e.storage.Resources().GetByName(task.WorkspaceID, task.ResourceName, models.ResourceTypeRobot)
	if err != nil {
		return "", fmt.Errorf("failed to load robot: %w", err)
	}

	// Parse robot spec
	parser := crd.NewParser()
	resource, err := parser.Parse([]byte(robotResource.Spec))
	if err != nil {
		return "", fmt.Errorf("failed to parse robot spec: %w", err)
	}

	robot, ok := resource.(*crd.Robot)
	if !ok {
		return "", fmt.Errorf("invalid robot resource")
	}

	// Load soul for system prompt
	soulResource, err := e.storage.Resources().GetByName(task.WorkspaceID, robot.Spec.Soul, models.ResourceTypeSoul)
	if err != nil {
		return "", fmt.Errorf("failed to load soul: %w", err)
	}

	soulDef, err := parser.Parse([]byte(soulResource.Spec))
	if err != nil {
		return "", fmt.Errorf("failed to parse soul: %w", err)
	}
	soul := soulDef.(*crd.Soul)

	// Load mind configuration
	mindResource, err := e.storage.Resources().GetByName(task.WorkspaceID, robot.Spec.Mind, models.ResourceTypeMind)
	if err != nil {
		return "", fmt.Errorf("failed to load mind: %w", err)
	}

	mindDef, err := parser.Parse([]byte(mindResource.Spec))
	if err != nil {
		return "", fmt.Errorf("failed to parse mind: %w", err)
	}
	mind := mindDef.(*crd.Mind)

	// Get or create LLM client
	client, err := e.getLLMClient(mind)
	if err != nil {
		return "", fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Build messages
	messages := []llm.Message{
		{
			Role:    "system",
			Content: soul.Spec.Personality,
		},
		{
			Role:    "user",
			Content: task.Prompt,
		},
	}

	// Report progress
	if callback != nil {
		callback(task.ID, 30, models.TaskStatusRunning, "Calling LLM...", nil)
	}

	// Call LLM with streaming
	var fullResponse string
	err = client.Stream(ctx, messages, func(chunk string) error {
		fullResponse += chunk
		if callback != nil {
			callback(task.ID, 60, models.TaskStatusRunning, chunk, map[string]interface{}{
				"type": "content",
			})
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	return fullResponse, nil
}

// executeTeam executes a team task
func (e *Executor) executeTeam(ctx context.Context, task *models.Task, callback ProgressCallback) (string, error) {
	// Load team resource
	teamResource, err := e.storage.Resources().GetByName(task.WorkspaceID, task.ResourceName, models.ResourceTypeTeam)
	if err != nil {
		return "", fmt.Errorf("failed to load team: %w", err)
	}

	// Parse team spec
	parser := crd.NewParser()
	resource, err := parser.Parse([]byte(teamResource.Spec))
	if err != nil {
		return "", fmt.Errorf("failed to parse team spec: %w", err)
	}

	team, ok := resource.(*crd.Team)
	if !ok {
		return "", fmt.Errorf("invalid team resource")
	}

	// Simple implementation: execute leader robot with member context
	// In a full implementation, this would coordinate between multiple agents

	var leaderResult string
	if team.Spec.Leader != "" {
		// Load leader robot
		leaderRobotResource, err := e.storage.Resources().GetByName(task.WorkspaceID, team.Spec.Leader, models.ResourceTypeRobot)
		if err != nil {
			return "", fmt.Errorf("failed to load leader robot: %w", err)
		}

		leaderRobotDef, err := parser.Parse([]byte(leaderRobotResource.Spec))
		if err != nil {
			return "", fmt.Errorf("failed to parse leader robot: %w", err)
		}
		leaderRobot := leaderRobotDef.(*crd.Robot)

		// Load leader's soul and mind
		soulResource, err := e.storage.Resources().GetByName(task.WorkspaceID, leaderRobot.Spec.Soul, models.ResourceTypeSoul)
		if err != nil {
			return "", fmt.Errorf("failed to load soul: %w", err)
		}
		soulDef, err := parser.Parse([]byte(soulResource.Spec))
		if err != nil {
			return "", fmt.Errorf("failed to parse soul: %w", err)
		}
		soul := soulDef.(*crd.Soul)

		mindResource, err := e.storage.Resources().GetByName(task.WorkspaceID, leaderRobot.Spec.Mind, models.ResourceTypeMind)
		if err != nil {
			return "", fmt.Errorf("failed to load mind: %w", err)
		}
		mindDef, err := parser.Parse([]byte(mindResource.Spec))
		if err != nil {
			return "", fmt.Errorf("failed to parse mind: %w", err)
		}
		mind := mindDef.(*crd.Mind)

		// Get LLM client
		client, err := e.getLLMClient(mind)
		if err != nil {
			return "", fmt.Errorf("failed to create LLM client: %w", err)
		}

		// Build team context
		teamContext := fmt.Sprintf("You are leading a team with %d members. Coordinate their work to accomplish the task.\n\nTeam members: %v\nCollaboration mode: %s",
			len(team.Spec.Members),
			getMemberNames(team),
			team.Spec.Mode,
		)

		messages := []llm.Message{
			{
				Role:    "system",
				Content: soul.Spec.Personality + "\n\n" + teamContext,
			},
			{
				Role:    "user",
				Content: task.Prompt,
			},
		}

		// Report progress
		if callback != nil {
			callback(task.ID, 30, models.TaskStatusRunning, "Team leader coordinating...", nil)
		}

		// Call LLM with streaming
		err = client.Stream(ctx, messages, func(chunk string) error {
			leaderResult += chunk
			if callback != nil {
				callback(task.ID, 70, models.TaskStatusRunning, chunk, map[string]interface{}{
					"type":  "content",
					"agent": "leader",
				})
			}
			return nil
		})

		if err != nil {
			return "", fmt.Errorf("leader execution failed: %w", err)
		}
	}

	return leaderResult, nil
}

// getLLMClient gets or creates an LLM client for a mind
func (e *Executor) getLLMClient(mind *crd.Mind) (llm.Client, error) {
	cacheKey := fmt.Sprintf("%s:%s:%s", mind.Spec.Provider, mind.Spec.ModelID, mind.Spec.BaseURL)

	if client, exists := e.llmClients[cacheKey]; exists {
		return client, nil
	}

	var client llm.Client
	apiKey := mind.Spec.APIKey

	switch mind.Spec.Provider {
	case "ollama":
		// Ollama runs locally, no API key needed
		baseURL := mind.Spec.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		client = llm.NewOllamaClient(mind.Spec.ModelID, baseURL)

	case "openai":
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("API key not configured for provider: %s", mind.Spec.Provider)
		}
		if mind.Spec.BaseURL != "" {
			client = llm.NewOpenAICompatibleClient(mind.Spec.ModelID, apiKey, mind.Spec.BaseURL)
		} else {
			client = llm.NewOpenAIClient(mind.Spec.ModelID, apiKey)
		}

	case "groq":
		// Groq is OpenAI-compatible with free tier
		if apiKey == "" {
			apiKey = os.Getenv("GROQ_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("API key not configured for provider: %s", mind.Spec.Provider)
		}
		client = llm.NewOpenAICompatibleClient(mind.Spec.ModelID, apiKey, "https://api.groq.com/openai/v1")

	case "together":
		// Together AI is OpenAI-compatible
		if apiKey == "" {
			apiKey = os.Getenv("TOGETHER_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("API key not configured for provider: %s", mind.Spec.Provider)
		}
		client = llm.NewOpenAICompatibleClient(mind.Spec.ModelID, apiKey, "https://api.together.xyz/v1")

	case "deepseek":
		// DeepSeek is OpenAI-compatible
		if apiKey == "" {
			apiKey = os.Getenv("DEEPSEEK_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("API key not configured for provider: %s", mind.Spec.Provider)
		}
		client = llm.NewOpenAICompatibleClient(mind.Spec.ModelID, apiKey, "https://api.deepseek.com/v1")

	case "openrouter":
		// OpenRouter aggregates many models
		if apiKey == "" {
			apiKey = os.Getenv("OPENROUTER_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("API key not configured for provider: %s", mind.Spec.Provider)
		}
		client = llm.NewOpenAICompatibleClient(mind.Spec.ModelID, apiKey, "https://openrouter.ai/api/v1")

	case "gemini", "google":
		// Google Gemini - has free tier
		if apiKey == "" {
			apiKey = os.Getenv("GEMINI_API_KEY")
		}
		if apiKey == "" {
			apiKey = os.Getenv("GOOGLE_API_KEY")
		}
		if apiKey == "" {
			return nil, fmt.Errorf("API key not configured for provider: %s", mind.Spec.Provider)
		}
		client = llm.NewGeminiClient(mind.Spec.ModelID, apiKey)

	default:
		// For unknown providers, try OpenAI-compatible if base_url is provided
		if mind.Spec.BaseURL != "" {
			client = llm.NewOpenAICompatibleClient(mind.Spec.ModelID, apiKey, mind.Spec.BaseURL)
		} else {
			return nil, fmt.Errorf("unsupported LLM provider: %s (provide base_url for OpenAI-compatible APIs)", mind.Spec.Provider)
		}
	}

	e.llmClients[cacheKey] = client
	return client, nil
}

// getMemberNames extracts member names from team
func getMemberNames(team *crd.Team) []string {
	return team.Spec.Members
}
