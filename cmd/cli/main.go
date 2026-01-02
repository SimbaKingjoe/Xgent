package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	apiURL   string
	token    string
	rootCmd  = &cobra.Command{
		Use:   "xgent-cli",
		Short: "Xgent-Go CLI tool",
		Long:  "Command line interface for Xgent-Go AI Agent platform",
	}
)

func main() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api", "http://localhost:8080", "API server URL")
	rootCmd.PersistentFlags().StringVar(&token, "token", os.Getenv("XGENT_TOKEN"), "API token")

	// Auth commands
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}
	authCmd.AddCommand(loginCmd, registerCmd)

	// Resource commands
	resourceCmd := &cobra.Command{
		Use:   "resource",
		Short: "Resource management commands",
	}
	resourceCmd.AddCommand(applyCmd, listResourcesCmd, getResourceCmd, deleteResourceCmd)

	// Task commands
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Task management commands",
	}
	taskCmd.AddCommand(createTaskCmd, listTasksCmd, getTaskCmd, logsCmd)

	// Workspace commands
	workspaceCmd := &cobra.Command{
		Use:   "workspace",
		Short: "Workspace management commands",
	}
	workspaceCmd.AddCommand(createWorkspaceCmd, listWorkspacesCmd)

	rootCmd.AddCommand(authCmd, resourceCmd, taskCmd, workspaceCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Auth commands
var loginCmd = &cobra.Command{
	Use:   "login [username] [password]",
	Short: "Login to Xgent",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := makeRequest("POST", "/api/v1/auth/login", map[string]string{
			"username": args[0],
			"password": args[1],
		}, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
			os.Exit(1)
		}

		var result map[string]interface{}
		json.Unmarshal(resp, &result)
		
		if tokenVal, ok := result["token"]; ok {
			fmt.Printf("Login successful!\n")
			fmt.Printf("Token: %s\n", tokenVal)
			fmt.Printf("\nSet environment variable:\n")
			fmt.Printf("export XGENT_TOKEN=%s\n", tokenVal)
		}
	},
}

var registerCmd = &cobra.Command{
	Use:   "register [username] [email] [password]",
	Short: "Register a new user",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := makeRequest("POST", "/api/v1/auth/register", map[string]string{
			"username": args[0],
			"email":    args[1],
			"password": args[2],
		}, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Registration failed: %v\n", err)
			os.Exit(1)
		}

		var result map[string]interface{}
		json.Unmarshal(resp, &result)
		fmt.Printf("Registration successful!\n")
		printJSON(resp)
	},
}

// Resource commands
var applyCmd = &cobra.Command{
	Use:   "apply -f [file]",
	Short: "Apply resources from YAML file",
	Run: func(cmd *cobra.Command, args []string) {
		file, _ := cmd.Flags().GetString("file")
		if file == "" {
			fmt.Fprintln(os.Stderr, "Error: -f flag is required")
			os.Exit(1)
		}

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read file: %v\n", err)
			os.Exit(1)
		}

		resp, err := makeRequest("POST", "/api/v1/resources/apply", string(data), token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Apply failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Resource applied successfully!\n")
		printJSON(resp)
	},
}

var listResourcesCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources",
	Run: func(cmd *cobra.Command, args []string) {
		resourceType, _ := cmd.Flags().GetString("type")
		path := "/api/v1/resources"
		if resourceType != "" {
			path += "?type=" + resourceType
		}

		resp, err := makeRequest("GET", path, nil, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "List failed: %v\n", err)
			os.Exit(1)
		}

		printJSON(resp)
	},
}

var getResourceCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get resource by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := makeRequest("GET", "/api/v1/resources/"+args[0], nil, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Get failed: %v\n", err)
			os.Exit(1)
		}

		printJSON(resp)
	},
}

var deleteResourceCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete resource by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := makeRequest("DELETE", "/api/v1/resources/"+args[0], nil, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Delete failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Resource deleted successfully!\n")
		printJSON(resp)
	},
}

// Task commands
var createTaskCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task",
	Run: func(cmd *cobra.Command, args []string) {
		title, _ := cmd.Flags().GetString("title")
		prompt, _ := cmd.Flags().GetString("prompt")
		resourceType, _ := cmd.Flags().GetString("resource-type")
		resourceName, _ := cmd.Flags().GetString("resource-name")

		if title == "" || prompt == "" || resourceType == "" || resourceName == "" {
			fmt.Fprintln(os.Stderr, "Error: --title, --prompt, --resource-type, and --resource-name are required")
			os.Exit(1)
		}

		resp, err := makeRequest("POST", "/api/v1/tasks", map[string]interface{}{
			"title":         title,
			"prompt":        prompt,
			"resource_type": resourceType,
			"resource_name": resourceName,
		}, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Create task failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Task created successfully!\n")
		printJSON(resp)
	},
}

var listTasksCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := makeRequest("GET", "/api/v1/tasks", nil, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "List failed: %v\n", err)
			os.Exit(1)
		}

		printJSON(resp)
	},
}

var getTaskCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get task by ID",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := makeRequest("GET", "/api/v1/tasks/"+args[0], nil, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Get failed: %v\n", err)
			os.Exit(1)
		}

		printJSON(resp)
	},
}

var logsCmd = &cobra.Command{
	Use:   "logs [task-id]",
	Short: "Get task logs",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := makeRequest("GET", "/api/v1/tasks/"+args[0]+"/logs", nil, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Get logs failed: %v\n", err)
			os.Exit(1)
		}

		printJSON(resp)
	},
}

// Workspace commands
var createWorkspaceCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new workspace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		description, _ := cmd.Flags().GetString("description")

		resp, err := makeRequest("POST", "/api/v1/workspaces", map[string]string{
			"name":        args[0],
			"description": description,
		}, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Create workspace failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Workspace created successfully!\n")
		printJSON(resp)
	},
}

var listWorkspacesCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspaces",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := makeRequest("GET", "/api/v1/workspaces", nil, token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "List failed: %v\n", err)
			os.Exit(1)
		}

		printJSON(resp)
	},
}

func init() {
	applyCmd.Flags().StringP("file", "f", "", "YAML file path")
	listResourcesCmd.Flags().String("type", "", "Resource type filter")

	createTaskCmd.Flags().String("title", "", "Task title")
	createTaskCmd.Flags().String("prompt", "", "Task prompt")
	createTaskCmd.Flags().String("resource-type", "", "Resource type (bot or team)")
	createTaskCmd.Flags().String("resource-name", "", "Resource name")

	createWorkspaceCmd.Flags().String("description", "", "Workspace description")
}

// Helper functions
func makeRequest(method, path string, body interface{}, authToken string) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		switch v := body.(type) {
		case string:
			reqBody = bytes.NewBufferString(v)
		default:
			jsonData, _ := json.Marshal(body)
			reqBody = bytes.NewBuffer(jsonData)
		}
	}

	req, err := http.NewRequest(method, apiURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	if body != nil {
		if _, ok := body.(string); ok && filepath.Ext(path) != "" {
			req.Header.Set("Content-Type", "text/yaml")
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func printJSON(data []byte) {
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		fmt.Println(string(data))
		return
	}

	formatted, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Println(string(data))
		return
	}

	fmt.Println(string(formatted))
}
