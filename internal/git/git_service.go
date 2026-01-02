package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// Service handles Git operations
type Service struct {
	workspaceDir string
	logger       *zap.Logger
}

// NewService creates a new Git service
func NewService(workspaceDir string, logger *zap.Logger) *Service {
	return &Service{
		workspaceDir: workspaceDir,
		logger:       logger,
	}
}

// CloneOptions contains options for cloning a repository
type CloneOptions struct {
	URL    string
	Branch string
	Depth  int
	Token  string // For private repositories
}

// Clone clones a Git repository
func (s *Service) Clone(opts CloneOptions, targetDir string) error {
	s.logger.Info("Cloning repository",
		zap.String("url", opts.URL),
		zap.String("branch", opts.Branch),
		zap.String("target", targetDir),
	)

	// Ensure target directory doesn't exist
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("target directory already exists: %s", targetDir)
	}

	// Build clone command
	args := []string{"clone"}

	// Add branch if specified
	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}

	// Add depth for shallow clone
	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
	}

	// Add URL with token if provided
	repoURL := opts.URL
	if opts.Token != "" {
		// Insert token into HTTPS URL (e.g., https://token@github.com/user/repo.git)
		if strings.HasPrefix(repoURL, "https://") {
			repoURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", opts.Token), 1)
		}
	}

	args = append(args, repoURL, targetDir)

	// Execute git clone
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error("Git clone failed",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	s.logger.Info("Repository cloned successfully", zap.String("target", targetDir))
	return nil
}

// CommitOptions contains options for committing changes
type CommitOptions struct {
	Message    string
	Files      []string // Files to add, empty means all
	AuthorName string
	AuthorEmail string
}

// Commit commits changes to a repository
func (s *Service) Commit(repoPath string, opts CommitOptions) error {
	s.logger.Info("Committing changes",
		zap.String("repo", repoPath),
		zap.String("message", opts.Message),
	)

	// Verify repo exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Add files
	addArgs := []string{"-C", repoPath, "add"}
	if len(opts.Files) > 0 {
		addArgs = append(addArgs, opts.Files...)
	} else {
		addArgs = append(addArgs, ".")
	}

	cmd := exec.Command("git", addArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %w, output: %s", err, string(output))
	}

	// Set author info if provided
	commitArgs := []string{"-C", repoPath, "commit", "-m", opts.Message}
	if opts.AuthorName != "" && opts.AuthorEmail != "" {
		commitArgs = append(commitArgs, "--author",
			fmt.Sprintf("%s <%s>", opts.AuthorName, opts.AuthorEmail))
	}

	// Commit
	cmd = exec.Command("git", commitArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error("Git commit failed",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return fmt.Errorf("git commit failed: %w, output: %s", err, string(output))
	}

	s.logger.Info("Changes committed successfully")
	return nil
}

// PushOptions contains options for pushing changes
type PushOptions struct {
	Remote string
	Branch string
	Token  string
	Force  bool
}

// Push pushes changes to remote repository
func (s *Service) Push(repoPath string, opts PushOptions) error {
	s.logger.Info("Pushing changes",
		zap.String("repo", repoPath),
		zap.String("remote", opts.Remote),
		zap.String("branch", opts.Branch),
	)

	// Set remote URL with token if provided
	if opts.Token != "" {
		remoteURL, err := s.GetRemoteURL(repoPath, opts.Remote)
		if err != nil {
			return err
		}

		if strings.HasPrefix(remoteURL, "https://") {
			authenticatedURL := strings.Replace(remoteURL, "https://", fmt.Sprintf("https://%s@", opts.Token), 1)
			if err := s.SetRemoteURL(repoPath, opts.Remote, authenticatedURL); err != nil {
				return err
			}
		}
	}

	// Build push command
	pushArgs := []string{"-C", repoPath, "push", opts.Remote}
	if opts.Branch != "" {
		pushArgs = append(pushArgs, opts.Branch)
	}
	if opts.Force {
		pushArgs = append(pushArgs, "--force")
	}

	// Execute push
	cmd := exec.Command("git", pushArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error("Git push failed",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return fmt.Errorf("git push failed: %w, output: %s", err, string(output))
	}

	s.logger.Info("Changes pushed successfully")
	return nil
}

// CreateBranch creates a new branch
func (s *Service) CreateBranch(repoPath, branchName string, checkout bool) error {
	args := []string{"-C", repoPath, "branch", branchName}
	cmd := exec.Command("git", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create branch: %w, output: %s", err, string(output))
	}

	if checkout {
		return s.CheckoutBranch(repoPath, branchName)
	}

	return nil
}

// CheckoutBranch checks out a branch
func (s *Service) CheckoutBranch(repoPath, branchName string) error {
	args := []string{"-C", repoPath, "checkout", branchName}
	cmd := exec.Command("git", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout branch: %w, output: %s", err, string(output))
	}
	return nil
}

// ListBranches lists all branches in a repository
func (s *Service) ListBranches(repoPath string) ([]string, error) {
	args := []string{"-C", repoPath, "branch", "--list"}
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w, output: %s", err, string(output))
	}

	// Parse branch list
	lines := strings.Split(string(output), "\n")
	var branches []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove asterisk from current branch
		branch := strings.TrimPrefix(line, "* ")
		branches = append(branches, branch)
	}

	return branches, nil
}

// GetStatus gets the repository status
func (s *Service) GetStatus(repoPath string) (string, error) {
	args := []string{"-C", repoPath, "status", "--short"}
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}
	return string(output), nil
}

// GetRemoteURL gets the URL of a remote
func (s *Service) GetRemoteURL(repoPath, remoteName string) (string, error) {
	args := []string{"-C", repoPath, "remote", "get-url", remoteName}
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// SetRemoteURL sets the URL of a remote
func (s *Service) SetRemoteURL(repoPath, remoteName, url string) error {
	args := []string{"-C", repoPath, "remote", "set-url", remoteName, url}
	cmd := exec.Command("git", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set remote URL: %w, output: %s", err, string(output))
	}
	return nil
}

// Diff gets the diff of changes
func (s *Service) Diff(repoPath string, files ...string) (string, error) {
	args := []string{"-C", repoPath, "diff"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}
	return string(output), nil
}

// Log gets commit logs
func (s *Service) Log(repoPath string, maxCount int) (string, error) {
	args := []string{"-C", repoPath, "log", "--oneline"}
	if maxCount > 0 {
		args = append(args, fmt.Sprintf("-n%d", maxCount))
	}
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get log: %w", err)
	}
	return string(output), nil
}

// Pull pulls changes from remote
func (s *Service) Pull(repoPath string, opts PushOptions) error {
	args := []string{"-C", repoPath, "pull", opts.Remote}
	if opts.Branch != "" {
		args = append(args, opts.Branch)
	}
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %w, output: %s", err, string(output))
	}
	return nil
}
