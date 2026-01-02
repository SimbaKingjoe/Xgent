# Contributing to Xgent-Go

Thank you for your interest in contributing to Xgent-Go! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions. We welcome contributors from all backgrounds and experience levels.

## How to Contribute

### Reporting Bugs

1. Check existing issues to avoid duplicates
2. Use the bug report template
3. Include:
   - Clear description of the problem
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (OS, Go version, etc.)

### Suggesting Features

1. Check existing issues/discussions
2. Describe the use case and benefits
3. Consider implementation complexity

### Pull Requests

1. **Fork** the repository
2. **Create a branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes** following our coding standards
4. **Write tests** for new functionality
5. **Run tests** to ensure nothing is broken:
   ```bash
   make test
   ```
6. **Commit** with clear messages:
   ```bash
   git commit -m "feat: add new feature description"
   ```
7. **Push** and create a Pull Request

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/xgent-go.git
cd xgent-go

# Add upstream remote
git remote add upstream https://github.com/xcode-ai/xgent-go.git

# Install dependencies
make deps
cd web && npm install && cd ..

# Run tests
make test
```

## Coding Standards

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run `go vet` before committing
- Write meaningful comments for exported functions
- Keep functions focused and small

### Frontend Code

- Use TypeScript with strict mode
- Follow React best practices
- Use functional components with hooks
- Format with Prettier

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation
- `style:` - Formatting
- `refactor:` - Code refactoring
- `test:` - Adding tests
- `chore:` - Maintenance

## Project Structure

```
xgent-go/
├── cmd/           # Entry points
├── internal/      # Private packages
├── pkg/           # Public packages
├── configs/       # Configuration
├── resources/     # CRD definitions
├── scripts/       # Utility scripts
└── web/           # Frontend
```

## Testing

- Write unit tests for new functions
- Maintain test coverage above 60%
- Use table-driven tests where appropriate

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./internal/api/...
```

## Documentation

- Update README.md for user-facing changes
- Add inline comments for complex logic
- Update API documentation for endpoint changes

## Questions?

Feel free to open an issue for any questions or discussions.

Thank you for contributing! 🙏
