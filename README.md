# Homelab Orchestration Platform

Unified multi-node orchestration for homelab infrastructure. Manages distributed systems (Raspberry Pis, servers, NAS) as a single cohesive platform.

## Overview

Multi-device homelab orchestration with intelligent resource management and automated deployment.

**Key capabilities:**
1. Automatic device analysis (RAM, storage, CPU)
2. Intelligent placement decisions
3. Resource sharing (shared database instances, auto-provisioning)
4. Zero-configuration deployment

**Differentiators:**
- Multi-node orchestration (unified resource management)
- Intelligent placement (automatic device selection)
- Resource sharing (database pooling, shared caching)
- Infrastructure-aware monitoring (aggregate metrics across devices)
- Zero-configuration deployment

**Features:**
- Unified dashboard (aggregate resources across devices)
- Smart deployment (automatic device selection)
- Database pooling (single shared instance, 60% RAM savings)
- Cross-device monitoring
- App marketplace (20+ curated recipes)

## Technical Architecture

**Design:**

- Agentless (SSH and Docker API)
- Intelligent scheduler (resource scoring algorithm)
- Database pooling (shared instances with auto-provisioning)
- Single binary (Go backend, embedded React frontend)
- Multi-network aware (VLAN/subnet support)
- Recipe-based (docker-compose.yaml + manifest.yaml)

**Status:**
- ✅ Device discovery and management
- ✅ Recipe-based marketplace (20+ apps)
- ✅ Single-device deployment
- 🚧 Intelligent resource scoring
- 🚧 Shared database infrastructure
- 🚧 Cross-device resource aggregation

Documentation: [docs/architecture.md](docs/architecture.md), [docs/mvp-vision.md](docs/mvp-vision.md)

## Quick Start

### Prerequisites

- Go 1.21+
- Node 18+
- Make

### Development

```bash
git clone https://github.com/jaredcannon/homelab-orchestration-platform
cd homelab-orchestration-platform
make install-deps
make dev
```

Services:
- Backend API: http://localhost:8080
- Frontend: http://localhost:5173

### Testing

```bash
make test              # All tests
make backend-test      # Backend only
```

### Building

```bash
make build            # Single binary with embedded frontend
./bin/homelab         # Run
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

## Current Status

Development environment functional. Core features in active development: device scanning, app marketplace, backup management.

Reference: [docs/architecture.md](docs/architecture.md)

## Tech Stack

**Backend**:
- Go 1.25.2 with Fiber v2
- GORM (being integrated)

**Frontend**:
- React 18 + TypeScript + Vite
- TanStack Query (being integrated)

## Contributing

Contributions welcome. Open issues or PRs for bugs, features, or improvements.

## License

MIT
