# PongHub - AI Agent Guide

This document provides guidance for AI agents (like GitHub Copilot, Claude, or other coding assistants) working on the PongHub project.

## Project Overview

PongHub is an open-source service status monitoring tool written in Go. It monitors services via HTTP requests, validates responses, checks SSL certificates, and generates static HTML reports deployed to GitHub Pages.

**Key Features:**

- Zero-intrusion monitoring
- Multi-port detection
- Intelligent response validation
- SSL certificate monitoring
- Real-time status display
- Notification system (email, webhook, GitHub Actions)

## Technical Stack

- **Language**: Go 1.24.5
- **Dependencies**: `gopkg.in/yaml.v3`, `github.com/google/uuid`
- **Build Tool**: Go toolchain (no additional build systems required)
- **Deployment**: GitHub Actions to GitHub Pages
- **Configuration**: YAML-based (`config.yaml`)

## Getting Started

### Prerequisites

- Go 1.24.5 installed
- Repository cloned locally

### Quick Setup

```bash
git clone https://github.com/WCY-dt/ponghub.git
cd ponghub
go mod download
```

### Basic Commands

```bash
# Build
go build -o bin/ponghub ./cmd/ponghub

# Test
go test ./...

# Run
./bin/ponghub
```

## Project Structure

```text
ponghub/
├── cmd/ponghub/          # Main application entry point
├── internal/             # Private Go packages
│   ├── checker/         # Service checking logic
│   ├── configure/       # Configuration handling
│   ├── logger/          # Log management
│   ├── notifier/        # Notification system
│   ├── reporter/        # HTML report generation
│   └── types/           # Type definitions
├── data/                # Generated output files
├── templates/           # HTML templates
├── static/              # Static assets (CSS, etc.)
├── config.yaml          # Configuration file
└── go.mod               # Go module definition
```

## Development Workflow

### Making Changes

1. **Understand the Change**: Review the relevant package in `internal/`
2. **Test Locally**: Run `go test ./...` and `./bin/ponghub`
3. **Verify Output**: Check `data/index.html` and `data/ponghub_log.json`
4. **Commit**: Ensure CI passes before merging

### Key Files to Know

- `cmd/ponghub/main.go`: Application entry point
- `internal/configure/config.go`: Configuration loading
- `internal/checker/services.go`: HTTP checking implementation
- `internal/reporter/report.go`: HTML generation
- `templates/report.html`: Report template
- `config.yaml`: Service configuration

## Configuration

Services are configured in `config.yaml`:

```yaml
services:
  - name: "My Service"
    endpoints:
      - url: "https://example.com/health"
        status_code: 200
        response_regex: "ok"
```

## Testing

- Unit tests exist for `internal/common/params`, `internal/notifier`, `internal/notifier/channels`
- Run tests with `go test ./...`
- Integration testing: Build and run, verify output files

## Deployment

- **Local**: Generates static files in `data/`
- **Production**: GitHub Actions deploys to GitHub Pages every 30 minutes
- **CI**: PR checks run tests and build verification

## Common Patterns

### Adding New Features

1. Identify the appropriate `internal/` package
2. Implement the logic
3. Update types if needed
4. Add tests
5. Update configuration if required

### Notification System

Supports multiple notification methods:

- Email via SMTP
- Webhooks with custom payloads
- GitHub Actions workflow failures

### Parameter System

PongHub supports dynamic parameters in configuration:

- `{{uuid}}`, `{{rand}}`, `{{env(VAR)}}`, etc.
- See `internal/common/params/` for implementation

## Best Practices for AI Agents

1. **Read the Code**: Start with `main.go` and follow the execution flow
2. **Test Changes**: Always run tests and build before suggesting changes
3. **Check Output**: Verify generated files after running
4. **Follow Go Conventions**: Use standard Go formatting and naming
5. **Update Documentation**: Modify README.md or comments as needed

## Troubleshooting

- **Build Issues**: Ensure Go 1.24.5 and run `go mod tidy`
- **Runtime Errors**: Check `config.yaml` syntax and network connectivity
- **Test Failures**: Review test output and fix logic issues
- **CI Failures**: Match local testing with CI environment

## Contributing

- Follow standard Go practices
- Add tests for new functionality
- Update documentation
- Ensure CI passes

For detailed build instructions and project layout, see `.github/copilot-instructions.md`.
