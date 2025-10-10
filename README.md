# Homelab Orchestration Platform

**Make self-hosting as simple as signing up for a SaaS product**

Version: 0.1.0 MVP (Phase 0 Complete)

## Philosophy

Inspired by Laravel Herd's approach to developer experience:
- Complexity hidden, not removed
- Convention over configuration
- One-click operations
- Beautiful, native-feeling UI
- Reversible actions
- Escape hatches everywhere

## Quick Start

### Prerequisites

- Go 1.21+
- Node 18+
- Make

### Development

```bash
# Clone the repository
git clone https://github.com/jaredcannon/homelab-orchestration-platform
cd homelab-orchestration-platform

# Install dependencies
make install-deps

# Run development servers
make dev
```

This starts:
- Backend API at http://localhost:8080
- Frontend at http://localhost:5173

### Testing

```bash
# Run backend tests
make test

# Run backend tests only
make backend-test
```

### Building

```bash
# Build single binary with embedded frontend
make build

# Run the binary
./bin/homelab
```

## Project Structure

```
homelab-orchestration-platform/
├── backend/           # Go backend (Fiber framework)
│   ├── cmd/server/   # Main entry point
│   ├── internal/     # Private application code
│   └── templates/    # docker-compose templates
├── frontend/          # React + TypeScript + Vite
│   └── src/          # Frontend source code
├── docs/             # Documentation
│   └── architecture.md
├── Makefile          # Build commands
└── Procfile          # Development server config
```

## Available Make Commands

```bash
make help              # Show all available commands
make dev               # Run development servers
make build             # Build production binary
make test              # Run all tests
make clean             # Clean build artifacts
make install-deps      # Install all dependencies
```

## Phase 0 Status ✅

**Complete!** Can run development servers with full hot-reload.

- [x] Monorepo structure
- [x] Go backend with Fiber
- [x] React frontend with Vite
- [x] Health check endpoint
- [x] Backend tests
- [x] API proxy configuration
- [x] Unified development workflow

**Next**: Phase 1 - Core Infrastructure (Device management, SSH, Docker validation)

## Documentation

See [docs/architecture.md](docs/architecture.md) for full architecture details.

## Tech Stack

**Backend**:
- Go 1.25.2
- Fiber v2 (web framework)
- GORM (will be added in Phase 1)

**Frontend**:
- React 18
- TypeScript
- Vite
- TanStack Query (will be added in Phase 1)

## Contributing

This project is in early development (Phase 0). See `docs/architecture.md` for the development roadmap.

## License

MIT
