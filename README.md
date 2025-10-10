# Homelab Orchestration Platform

A centralized dashboard for managing your homelab without the headache.

> **Note:** This is a personal project I'm building as I set up my own homelab. It's in early development and things will change as I figure out what works. Contributions and feedback are welcome!

## What is this?

Managing a homelab shouldn't require a PhD in DevOps. This project aims to make self-hosting as simple as signing up for a SaaS product - automatically discover devices on your network, get opinionated recommendations for what to run where, and deploy apps without diving into config files.

**Key ideas:**
- **Auto-discovery**: Scan your network and detect servers, NAS devices, Raspberry Pis, etc.
- **Smart recommendations**: Get app suggestions based on each device's available resources
- **Opinionated setup**: Sensible defaults that just work, with escape hatches when you need them
- **App marketplace**: Browse and deploy open-source apps with one click
- **Centralized view**: See your entire homelab at a glance
- **Encrypted backups**: Backup to cloud providers or local network storage
- **Low barrier to entry**: You shouldn't need to be a sysadmin to host your own services

The philosophy is simple: hide complexity, don't remove it. Everything should be reversible, and you should always have an escape hatch to the underlying tools if you need them.

## How is this different from Coolify/Portainer/etc?

This isn't trying to replace those tools. Instead, it's focused on the broader homelab experience:

- **Multi-device orchestration**: Most tools focus on a single server. This is built for managing multiple devices at once (your main server, that old laptop, a couple of Raspberry Pis, etc.)
- **Discovery-first**: Automatically find what's on your network instead of manually adding everything
- **Resource-aware**: Smart recommendations about which device should run what based on available CPU/RAM/storage
- **Homelab-specific features**: Built-in encrypted backups, network-wide monitoring, and other features that matter when you're running your own infrastructure
- **Opinionated by default**: Less "here are 50 config options" and more "this will just work, but you can change it if you want"

Think of it as the layer above your deployment tools - helping you figure out what to run where, keeping everything backed up, and giving you a single pane of glass for your entire setup.

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

## Current Status

The foundation is in place - you can spin up the dev environment and start poking around. Core features like device scanning, the app marketplace, and backup management are being actively developed.

Check [docs/architecture.md](docs/architecture.md) for more technical details on how things are structured.

## Tech Stack

**Backend**:
- Go 1.25.2 with Fiber v2
- GORM (being integrated)

**Frontend**:
- React 18 + TypeScript + Vite
- TanStack Query (being integrated)

## Contributing

This is a personal project that's being actively developed, but I'm open to contributions! If you want to help out or have ideas, feel free to open an issue or PR. Just keep in mind that things are still pretty fluid as I nail down the exact direction.

## License

MIT
