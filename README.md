# Xgent-Go

<p align="center">
  <strong>🚀 High-Performance AI Agent Orchestration Platform</strong>
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#architecture">Architecture</a> •
  <a href="#documentation">Documentation</a> •
  <a href="#contributing">Contributing</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg" alt="PRs Welcome">
</p>

---

Xgent-Go is an enterprise-grade AI Agent orchestration platform built with pure Golang. It provides a complete solution for defining, organizing, and running intelligent agent teams with a Kubernetes-inspired CRD (Custom Resource Definition) design.

## ✨ Features

- **🚀 Pure Golang** - High performance, low memory footprint, single binary deployment
- **🎨 CRD Resource System** - Kubernetes-style declarative configuration for agents
- **🤖 Multi-Agent Orchestration** - Support for Coordinate, Collaborate, and Route collaboration modes
- **🔧 Extensible Executors** - Built-in Agno executor with Python bridge, extensible for other engines
- **🐳 Docker Sandbox** - Isolated execution environment with resource limits
- **🔐 Enterprise Features** - User authentication, multi-tenancy, access control
- **📊 Real-time Monitoring** - WebSocket-based task progress and log streaming
- **🔗 Git Integration** - Deep integration with GitHub/GitLab

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Frontend (Next.js)                        │
└────────────────────────────┬────────────────────────────────┘
                             │ HTTP/WebSocket
┌────────────────────────────▼────────────────────────────────┐
│                    Backend API (Gin)                         │
│   Routes • Middlewares • WebSocket Streaming                 │
└────────────────────────────┬────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────┐
│                 Task Orchestrator                            │
│   Task Queue • Worker Pool • State Management                │
└────────────────────────────┬────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────┐
│                 CRD Resource Manager                         │
│   YAML Parser • Validator • Ghost/Model/Shell/Bot/Team       │
└────────────────────────────┬────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────┐
│                 Executor Layer                               │
│   Agno Executor (Python Bridge) • Docker Executor            │
└────────────────────────────┬────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────┐
│                    Storage Layer                             │
│   MySQL/PostgreSQL (GORM) • Redis (Cache) • MinIO/S3        │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+ (for frontend)
- MySQL 8.0+
- Python 3.10+ (for Agno executor)
- Redis (optional)

### Installation

```bash
# Clone the repository
git clone https://github.com/xcode-ai/xgent-go.git
cd xgent-go

# Install Go dependencies
make deps

# Install Python dependencies (for Agno executor)
pip install agno openai anthropic

# Install frontend dependencies
cd web && npm install && cd ..
```

### Configuration

```bash
# Copy example config
cp configs/config.yaml configs/config.local.yaml

# Edit config with your settings
vim configs/config.local.yaml
```

Required environment variables:
```bash
export DB_PASSWORD=your_database_password
export OPENAI_API_KEY=your_openai_key        # Optional
export ANTHROPIC_API_KEY=your_anthropic_key  # Optional
```

### Running

```bash
# Start MySQL (if using Docker)
docker-compose up -d mysql

# Start the API server
make run-server

# In another terminal, start the frontend
cd web && npm run dev
```

### Access

| Service | URL |
|---------|-----|
| API Server | http://localhost:8080 |
| Health Check | http://localhost:8080/health |
| Web UI | http://localhost:3001 |

## 📖 Core Concepts

### CRD Resource Types

| Resource | Description |
|----------|-------------|
| **Soul** | Agent personality and system prompt |
| **Mind** | LLM configuration (OpenAI/Claude/Gemini) |
| **Craft** | Tools and capabilities (MCP Tools) |
| **Robot** | Agent instance (Soul + Mind + Craft) |
| **Team** | Multi-agent collaboration group |

### Collaboration Modes

- **Coordinate** - Leader assigns tasks to specific members
- **Collaborate** - All members work in parallel
- **Route** - Select the most suitable member for execution

### Example: Creating a Robot

```yaml
# resources/examples/my-robot.yaml
apiVersion: xgent.ai/v1
kind: Robot
metadata:
  name: code-reviewer
  description: Expert code reviewer
spec:
  soul: code-review-expert
  mind: gpt-4
  craft: coding-tools
```

## 📁 Project Structure

```
xgent-go/
├── cmd/
│   ├── server/          # API server
│   ├── worker/          # Task executor
│   └── cli/             # CLI tool
├── internal/
│   ├── api/             # API routes and handlers
│   ├── orchestrator/    # Task scheduling
│   ├── crd/             # CRD resource system
│   ├── executor/        # Execution engines
│   ├── storage/         # Database layer
│   └── git/             # Git integration
├── pkg/                 # Public packages
├── configs/             # Configuration files
├── resources/           # CRD resource definitions
├── scripts/             # Utility scripts
├── web/                 # Frontend (Next.js)
├── Makefile
└── docker-compose.yml
```

## 🔧 Development

### Build

```bash
# Build all binaries
make build

# Build specific components
make build-server
make build-worker
make build-cli
```

### Test

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

### Docker

```bash
# Build Docker image
docker build -t xgent-go:latest .

# Run with Docker Compose
docker-compose up -d
```

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) before submitting a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Agno](https://github.com/agno-agi/agno) - AI Agent framework
- [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [GORM](https://gorm.io/) - ORM library for Go

---

<p align="center">
  Made with ❤️ by the Xgent Team
</p>
