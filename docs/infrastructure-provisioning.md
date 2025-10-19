# Infrastructure Provisioning: Laravel Forge for Homelabs

## Overview

Inspired by Laravel Forge and Vapor, our infrastructure provisioning system provides one-click deployment of complete server environments. Instead of manually installing PHP, Nginx, databases, and dependencies, users click "Provision App Server" and get a production-ready Laravel environment in minutes.

## Philosophy

**Laravel Forge** revolutionized PHP deployment by abstracting server provisioning:
- Click "Provision Server" → Get Nginx, PHP, MySQL, Redis, Node
- Click "Deploy Site" → Laravel app goes live
- Zero terminal commands required

**Our System** brings this experience to homelabs:
- Multi-device orchestration (Forge manages single servers)
- Intelligent device selection (our unique advantage)
- Self-hosted (no monthly fees)
- Broader than Laravel (but Laravel-first)

## Comparison: Forge vs Our Platform

| Feature | Laravel Forge | Laravel Vapor | Our Platform |
|---------|---------------|---------------|--------------|
| Target | Single VPS | AWS Lambda | Homelab (multi-device) |
| Provisioning | DigitalOcean, Linode, AWS | AWS only | Any device via SSH |
| Server Types | App, Web, Database | Serverless (no servers) | App, Web, Database, Worker |
| Database | Standalone MySQL/Postgres | RDS, Aurora Serverless | Shared instances (resource efficient) |
| Queue Workers | Supervisor | SQS | Supervisor (auto-configured) |
| Scheduler | Cron | CloudWatch Events | Cron (auto-configured) |
| Redis | Standalone instance | ElastiCache | Shared instance (pooled) |
| Cost | $12/mo + server costs | $0 + AWS usage | $0 (own hardware) |
| Multi-Server | Manual linking | N/A | Automatic (intelligent placement) |
| PHP Versions | Multiple (7.4, 8.0, 8.1, etc.) | Configurable | Multiple (configurable) |

## Server Types

Inspired by Forge's server types, we provide specialized infrastructure templates:

### 1. App Server

**Purpose**: Complete Laravel application environment

**Includes:**
- Nginx web server
- PHP 8.3 (or configurable version)
- Composer
- Node.js + npm
- PostgreSQL or MySQL database
- Redis cache
- Supervisor (queue workers)
- Cron (Laravel scheduler)

**Use Case**: Deploy full-stack Laravel application

**Configuration:**
```yaml
# Server Type: App Server
type: laravel-app-server

components:
  - nginx
  - php-8.3-fpm
  - composer
  - nodejs-20
  - postgresql-15
  - redis-7
  - supervisor
  - cron

auto_configure:
  - php.ini (memory_limit, upload_max_filesize)
  - nginx.conf (Laravel-optimized)
  - supervisor (queue:work configuration)
  - crontab (Laravel scheduler)
```

### 2. Web Server

**Purpose**: Frontend-only server (no database)

**Includes:**
- Nginx
- PHP 8.3
- Composer
- Node.js + npm
- Redis (optional)

**Use Case**: Stateless web tier, network to separate database server

**Configuration:**
```yaml
# Server Type: Web Server
type: laravel-web-server

components:
  - nginx
  - php-8.3-fpm
  - composer
  - nodejs-20
  - redis-7 (optional)

auto_configure:
  - nginx.conf (Laravel-optimized, no DB)
  - php.ini (optimized for web processing)
```

### 3. Database Server

**Purpose**: Dedicated database server

**Includes:**
- PostgreSQL or MySQL
- Automated backups
- Performance tuning
- Remote access configuration

**Use Case**: Centralized database for multiple app servers

**Configuration:**
```yaml
# Server Type: Database Server
type: database-server

components:
  - postgresql-15
  # OR mysql-8.0

auto_configure:
  - postgresql.conf (homelab-optimized)
  - pg_hba.conf (allow remote connections)
  - automated_backups (daily at 2 AM)
  - replication (optional, for HA)
```

### 4. Worker Server

**Purpose**: Background job processing

**Includes:**
- PHP 8.3
- Supervisor
- Redis client
- Horizon (Laravel queue dashboard)

**Use Case**: Dedicated server for Laravel queues

**Configuration:**
```yaml
# Server Type: Worker Server
type: laravel-worker-server

components:
  - php-8.3-cli
  - supervisor
  - redis-cli

auto_configure:
  - supervisor (queue:work, multiple workers)
  - horizon (Laravel queue dashboard)
  - cron (queue:restart on deploy)
```

### 5. Cache Server

**Purpose**: Dedicated Redis/Memcached server

**Includes:**
- Redis or Memcached
- Performance tuning
- Persistence configuration

**Use Case**: Shared cache for multiple app servers

**Configuration:**
```yaml
# Server Type: Cache Server
type: cache-server

components:
  - redis-7
  # OR memcached

auto_configure:
  - redis.conf (maxmemory, eviction policies)
  - persistence (RDB snapshots)
  - remote_access (allow network connections)
```

## Provisioning Workflow

### User Experience: Provision Laravel App Server

**Step 1: Select Server Type**
```
┌────────────────────────────────────────────────────┐
│                                                    │
│  Provision Infrastructure                         │
│                                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────┐ │
│  │              │  │              │  │          │ │
│  │  App Server  │  │  Web Server  │  │ Database │ │
│  │              │  │              │  │  Server  │ │
│  │  Full stack  │  │ Frontend only│  │  DB only │ │
│  │  Laravel env │  │  No database │  │  Shared  │ │
│  │              │  │              │  │          │ │
│  └──────────────┘  └──────────────┘  └──────────┘ │
│                                                    │
│  ┌──────────────┐  ┌──────────────┐               │
│  │              │  │              │               │
│  │ Worker Server│  │ Cache Server │               │
│  │              │  │              │               │
│  │ Queue jobs   │  │ Redis/Memcached│             │
│  │  Horizon     │  │   Shared     │               │
│  │              │  │              │               │
│  └──────────────┘  └──────────────┘               │
│                                                    │
└────────────────────────────────────────────────────┘
```

**Step 2: Configure Server**
```
┌────────────────────────────────────────────────────┐
│                                                    │
│  Configure Laravel App Server                     │
│                                                    │
│  Server Name                                       │
│  ┌────────────────────────────────────────────┐   │
│  │ laravel-production                         │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  PHP Version                                       │
│  ┌────────────────────────────────────────────┐   │
│  │ PHP 8.3 (recommended)                   ▼ │   │
│  └────────────────────────────────────────────┘   │
│    Options: 8.0, 8.1, 8.2, 8.3                   │
│                                                    │
│  Database Engine                                   │
│  ┌────────────────────────────────────────────┐   │
│  │ PostgreSQL 15 (recommended)             ▼ │   │
│  └────────────────────────────────────────────┘   │
│    Options: PostgreSQL 15, MySQL 8.0, MariaDB    │
│                                                    │
│  ☑ Include Redis cache                           │
│  ☑ Include Supervisor (queue workers)            │
│  ☑ Configure Laravel scheduler (cron)            │
│  ☑ Install Composer                              │
│  ☑ Install Node.js 20                            │
│                                                    │
│  [Cancel]                    [Provision Server]   │
│                                                    │
└────────────────────────────────────────────────────┘
```

**Step 3: Device Selection** (Automatic with Override)
```
┌────────────────────────────────────────────────────┐
│                                                    │
│  🔍 Analyzing your homelab...                     │
│                                                    │
│  ✅ Recommended: homelab-server-02 (Score: 94/100)│
│     • 12GB RAM available (app needs 4GB)         │
│     • 200GB SSD storage free                      │
│     • Current load: 35% (plenty of headroom)     │
│     • Uptime: 99.9%                               │
│                                                    │
│  Override?  [Select different device ▼]           │
│     • homelab-server-01: Score 78 (8GB RAM, 75% load)│
│     • homelab-pi-01: Score 45 (4GB RAM, SD card) │
│                                                    │
│  [Deploy to homelab-server-02]                    │
│                                                    │
└────────────────────────────────────────────────────┘
```

**Step 4: Real-Time Provisioning**
```
┌────────────────────────────────────────────────────┐
│                                                    │
│  Provisioning Laravel App Server...               │
│                                                    │
│  ✅ Installing Nginx (1/10) - 18 seconds          │
│  ✅ Installing PHP 8.3 (2/10) - 42 seconds        │
│  ✅ Installing Composer (3/10) - 8 seconds        │
│  ✅ Installing Node.js 20 (4/10) - 35 seconds     │
│  ✅ Installing PostgreSQL 15 (5/10) - 55 seconds  │
│  ✅ Installing Redis 7 (6/10) - 15 seconds        │
│  ✅ Installing Supervisor (7/10) - 12 seconds     │
│  ✅ Configuring Nginx (8/10) - 5 seconds          │
│  ✅ Configuring PHP-FPM (9/10) - 3 seconds        │
│  ⏳ Configuring Laravel scheduler (10/10)...      │
│                                                    │
│  Progress: ████████████████████░░ 90%             │
│                                                    │
└────────────────────────────────────────────────────┘
```

**Step 5: Success**
```
┌────────────────────────────────────────────────────┐
│                                                    │
│  ✅ Laravel App Server Ready!                     │
│                                                    │
│  Server: laravel-production                       │
│  Device: homelab-server-02                        │
│  IP Address: 192.168.1.105                        │
│                                                    │
│  Installed Components:                            │
│  • Nginx 1.24.0                                   │
│  • PHP 8.3.11                                     │
│  • PostgreSQL 15.4                                │
│  • Redis 7.2.3                                    │
│  • Composer 2.6.5                                 │
│  • Node.js 20.9.0                                 │
│  • Supervisor 4.2.5                               │
│                                                    │
│  Database Credentials:                            │
│  Host: localhost                                  │
│  Database: laravel_production                     │
│  User: laravel_user                               │
│  Password: [copied to clipboard]                  │
│                                                    │
│  Next Steps:                                      │
│  1. Deploy your Laravel app                       │
│  2. Configure environment variables               │
│  3. Run migrations                                │
│                                                    │
│  [Deploy Laravel App]  [View Server Details]      │
│                                                    │
└────────────────────────────────────────────────────┘
```

## Database Provisioning

### One-Click Database Creation

Similar to Laravel Vapor's database provisioning:

```
┌────────────────────────────────────────────────────┐
│                                                    │
│  Provision Database                               │
│                                                    │
│  Database Engine                                   │
│  ┌────────────────────────────────────────────┐   │
│  │ PostgreSQL 15                           ▼ │   │
│  └────────────────────────────────────────────┘   │
│    Options: PostgreSQL 15, MySQL 8.0, MariaDB 10 │
│                                                    │
│  Deployment Type                                   │
│  ┌────────────────────────────────────────────┐   │
│  │ Shared Instance (Recommended)           ▼ │   │
│  └────────────────────────────────────────────┘   │
│    Options:                                       │
│    • Shared Instance (uses existing, saves RAM)  │
│    • Dedicated Instance (new container)          │
│                                                    │
│  Database Name                                     │
│  ┌────────────────────────────────────────────┐   │
│  │ myapp_production                          │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  Auto-generate credentials? ☑                    │
│                                                    │
│  [Cancel]                        [Create Database]│
│                                                    │
└────────────────────────────────────────────────────┘
```

**Result:**
```
✅ Database Created: myapp_production

Connection String:
postgresql://myapp_user:g3n3r4t3d_p4ssw0rd@192.168.1.105:5432/myapp_production

Laravel .env format:
DB_CONNECTION=pgsql
DB_HOST=192.168.1.105
DB_PORT=5432
DB_DATABASE=myapp_production
DB_USERNAME=myapp_user
DB_PASSWORD=g3n3r4t3d_p4ssw0rd

[Copy to Clipboard]  [Use in App Deployment]
```

### Database Types

**Shared Instance** (Default, Recommended)
- Uses existing PostgreSQL/MySQL container
- Creates new database within shared instance
- Saves ~500MB-1GB RAM per database
- Automatic backup included
- Suitable for most applications

**Dedicated Instance**
- New isolated database container
- Full control over configuration
- Required for high-load databases
- More RAM usage (~1GB minimum)

## Laravel App Deployment

### From Git Repository

```
┌────────────────────────────────────────────────────┐
│                                                    │
│  Deploy Laravel Application                       │
│                                                    │
│  Git Repository                                    │
│  ┌────────────────────────────────────────────┐   │
│  │ https://github.com/user/my-laravel-app    │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  Branch                                            │
│  ┌────────────────────────────────────────────┐   │
│  │ main                                       │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  Deploy to Server                                  │
│  ┌────────────────────────────────────────────┐   │
│  │ laravel-production                      ▼ │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  Domain                                            │
│  ┌────────────────────────────────────────────┐   │
│  │ app.homelab.local                         │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  Database                                          │
│  ┌────────────────────────────────────────────┐   │
│  │ myapp_production (auto-detected)        ▼ │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  ☑ Run composer install                          │
│  ☑ Run npm install && npm run build              │
│  ☑ Run migrations (php artisan migrate --force)  │
│  ☑ Clear caches                                   │
│  ☑ Configure queue workers (Supervisor)          │
│  ☑ Configure scheduler (Cron)                    │
│                                                    │
│  [Cancel]                              [Deploy]    │
│                                                    │
└────────────────────────────────────────────────────┘
```

### Deployment Process

```go
type LaravelDeployment struct {
    GitRepo       string
    Branch        string
    ServerID      uuid.UUID
    Domain        string
    DatabaseID    uuid.UUID
    PHPVersion    string

    Steps []DeploymentStep
}

type DeploymentStep struct {
    Name        string
    Command     string
    Status      string
    Output      string
    Duration    time.Duration
}

func (s *InfrastructureService) DeployLaravelApp(deployment *LaravelDeployment) error {
    steps := []DeploymentStep{
        {Name: "Clone repository", Command: "git clone ..."},
        {Name: "Install Composer dependencies", Command: "composer install --no-dev --optimize-autoloader"},
        {Name: "Install NPM dependencies", Command: "npm ci"},
        {Name: "Build assets", Command: "npm run build"},
        {Name: "Generate app key", Command: "php artisan key:generate --force"},
        {Name: "Run migrations", Command: "php artisan migrate --force"},
        {Name: "Clear caches", Command: "php artisan config:cache && php artisan route:cache"},
        {Name: "Configure queue workers", Command: "supervisorctl reread && supervisorctl update"},
        {Name: "Configure Nginx vhost", Command: "nginx -t && systemctl reload nginx"},
        {Name: "Health check", Command: "curl http://localhost/health"},
    }

    for i, step := range steps {
        s.broadcastProgress(deployment.ID, i+1, len(steps), step.Name)

        err := s.executeStep(deployment, &step)
        if err != nil {
            s.rollback(deployment, i)
            return err
        }
    }

    return nil
}
```

## Infrastructure Templates

### Recipe Format for Infrastructure

Infrastructure components are recipes too:

```yaml
# marketplace-recipes/infrastructure/laravel-app-server/manifest.yaml

id: laravel-app-server
name: "Laravel App Server"
category: infrastructure
tagline: "Complete Laravel production environment"
description: "Nginx, PHP 8.3, PostgreSQL, Redis, Supervisor, and Cron pre-configured for Laravel applications"

icon_url: "https://cdn.example.com/laravel-server.png"

# This is an infrastructure recipe
is_infrastructure: true
server_type: "app_server"

requirements:
  memory:
    minimum: 2GB
    recommended: 4GB
  storage:
    minimum: 20GB
    recommended: 50GB
  cpu:
    minimum_cores: 2
    recommended_cores: 4

# Components to install
components:
  - name: nginx
    version: "latest"
    config_template: "nginx.conf.tmpl"

  - name: php
    version: "8.3"
    extensions:
      - fpm
      - cli
      - mbstring
      - xml
      - pgsql
      - mysql
      - redis
      - curl
      - zip
      - gd
      - intl
      - bcmath
    config_template: "php.ini.tmpl"

  - name: composer
    version: "2"
    global_packages:
      - laravel/installer

  - name: nodejs
    version: "20"
    global_packages:
      - npm
      - yarn

  - name: postgresql
    version: "15"
    config_template: "postgresql.conf.tmpl"
    create_user: true
    create_database: true

  - name: redis
    version: "7"
    config_template: "redis.conf.tmpl"

  - name: supervisor
    version: "latest"
    config_template: "supervisor.conf.tmpl"

# Configuration options
config_options:
  - name: php_version
    label: "PHP Version"
    type: select
    options: ["8.0", "8.1", "8.2", "8.3"]
    default: "8.3"

  - name: database_engine
    label: "Database Engine"
    type: select
    options: ["postgresql", "mysql", "mariadb"]
    default: "postgresql"

  - name: include_redis
    label: "Include Redis"
    type: boolean
    default: true

  - name: include_supervisor
    label: "Include Supervisor (Queue Workers)"
    type: boolean
    default: true

  - name: include_nodejs
    label: "Include Node.js"
    type: boolean
    default: true

# Post-installation setup
post_install:
  - type: command
    command: "systemctl enable nginx php8.3-fpm postgresql redis supervisor"

  - type: command
    command: "ufw allow 80/tcp && ufw allow 443/tcp"

  - type: message
    title: "Laravel App Server Ready"
    message: |
      Your Laravel app server is configured and ready.

      Next steps:
      1. Deploy your Laravel application
      2. Configure your domain in Nginx
      3. Set up SSL with Let's Encrypt

      Server details:
      - Nginx: Listening on ports 80, 443
      - PHP: Version ${PHP_VERSION}, PHP-FPM enabled
      - Database: ${DATABASE_ENGINE} installed
      - Redis: Running on port 6379
      - Supervisor: Queue workers ready
```

### Template Files

**nginx.conf.tmpl:**
```nginx
server {
    listen 80;
    listen [::]:80;
    server_name ${DOMAIN};

    root /var/www/${APP_NAME}/public;
    index index.php;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/var/run/php/php${PHP_VERSION}-fpm.sock;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;
    }

    location ~ /\.(?!well-known).* {
        deny all;
    }
}
```

**supervisor.conf.tmpl:**
```ini
[program:${APP_NAME}-worker]
process_name=%(program_name)s_%(process_num)02d
command=php /var/www/${APP_NAME}/artisan queue:work --sleep=3 --tries=3 --max-time=3600
autostart=true
autorestart=true
stopasgroup=true
killasgroup=true
user=${APP_USER}
numprocs=4
redirect_stderr=true
stdout_logfile=/var/www/${APP_NAME}/storage/logs/worker.log
stopwaitsecs=3600

[program:${APP_NAME}-scheduler]
command=php /var/www/${APP_NAME}/artisan schedule:run
autostart=true
autorestart=true
user=${APP_USER}
redirect_stderr=true
stdout_logfile=/var/www/${APP_NAME}/storage/logs/scheduler.log
```

## Queue Worker Management

### Horizon Support

Automatically configure Laravel Horizon for queue management:

```yaml
# When deploying Laravel app with Horizon

post_install:
  - type: command
    title: "Installing Horizon"
    command: |
      cd /var/www/${APP_NAME}
      composer require laravel/horizon
      php artisan horizon:install
      php artisan horizon:publish

  - type: supervisor_config
    title: "Configuring Horizon Supervisor"
    config: |
      [program:${APP_NAME}-horizon]
      process_name=%(program_name)s
      command=php /var/www/${APP_NAME}/artisan horizon
      autostart=true
      autorestart=true
      user=${APP_USER}
      redirect_stderr=true
      stdout_logfile=/var/www/${APP_NAME}/storage/logs/horizon.log
      stopwaitsecs=3600

  - type: message
    title: "Horizon Configured"
    message: |
      Laravel Horizon is installed and configured.
      Access the dashboard at: https://${DOMAIN}/horizon
```

## Scheduler Configuration

Automatically configure Laravel's task scheduler:

```bash
# Added to crontab during provisioning

* * * * * cd /var/www/${APP_NAME} && php artisan schedule:run >> /dev/null 2>&1
```

## Multi-Environment Support

### Environment Management

```
┌────────────────────────────────────────────────────┐
│                                                    │
│  Environments                                      │
│                                                    │
│  ┌────────────────────────────────────────────┐   │
│  │                                            │   │
│  │  Production                                │   │
│  │  laravel-production @ homelab-server-02    │   │
│  │  Status: ✅ Running                        │   │
│  │  https://app.homelab.local                 │   │
│  │                                            │   │
│  │  [View] [Deploy] [Logs]                    │   │
│  │                                            │   │
│  ├────────────────────────────────────────────┤   │
│  │                                            │   │
│  │  Staging                                   │   │
│  │  laravel-staging @ homelab-server-01       │   │
│  │  Status: ✅ Running                        │   │
│  │  https://staging.homelab.local             │   │
│  │                                            │   │
│  │  [View] [Deploy] [Logs]                    │   │
│  │                                            │   │
│  ├────────────────────────────────────────────┤   │
│  │                                            │   │
│  │  Development                               │   │
│  │  Not provisioned                           │   │
│  │                                            │   │
│  │  [Create Development Environment]          │   │
│  │                                            │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
└────────────────────────────────────────────────────┘
```

## Resource Optimization

### Shared vs Dedicated Resources

**Forge Approach**: Each site gets dedicated resources
- Site A: Nginx + PHP + MySQL = 1.5GB RAM
- Site B: Nginx + PHP + MySQL = 1.5GB RAM
- Total: 3GB RAM for 2 sites

**Our Approach**: Shared infrastructure
- Shared Nginx (all sites)
- Shared PHP-FPM pool (configurable workers)
- Shared PostgreSQL instance (multiple databases)
- Shared Redis (separate key prefixes)
- Total: 1.2GB RAM for 2 sites (60% reduction)

### Configuration

```yaml
# Shared infrastructure configuration

nginx:
  shared: true
  sites:
    - domain: app1.local
      root: /var/www/app1/public
    - domain: app2.local
      root: /var/www/app2/public

php_fpm:
  shared: true
  pool_config:
    pm: dynamic
    pm_max_children: 20
    pm_start_servers: 5
    pm_min_spare_servers: 2
    pm_max_spare_servers: 10

postgresql:
  shared: true
  databases:
    - name: app1_production
      user: app1_user
    - name: app2_production
      user: app2_user

redis:
  shared: true
  key_prefix_per_app: true
```

## API Design

### Provisioning Endpoints

```
# Provision server type
POST /api/v1/infrastructure/provision
{
  "server_type": "laravel-app-server",
  "name": "laravel-production",
  "device_id": "uuid",
  "config": {
    "php_version": "8.3",
    "database_engine": "postgresql",
    "include_redis": true,
    "include_supervisor": true
  }
}

# Create database
POST /api/v1/infrastructure/databases
{
  "engine": "postgresql",
  "name": "myapp_production",
  "deployment_type": "shared",  # or "dedicated"
  "device_id": "uuid",
  "auto_credentials": true
}

# Deploy Laravel app
POST /api/v1/infrastructure/laravel/deploy
{
  "git_repo": "https://github.com/user/app",
  "branch": "main",
  "server_id": "uuid",
  "domain": "app.homelab.local",
  "database_id": "uuid",
  "run_migrations": true,
  "build_assets": true,
  "configure_queues": true
}

# List infrastructure
GET /api/v1/infrastructure/servers
GET /api/v1/infrastructure/databases
GET /api/v1/infrastructure/apps

# Manage app
POST /api/v1/infrastructure/apps/:id/redeploy
POST /api/v1/infrastructure/apps/:id/rollback
GET  /api/v1/infrastructure/apps/:id/logs
POST /api/v1/infrastructure/apps/:id/env
```

## Comparison with Competitors

| Feature | Laravel Forge | Coolify | Dokploy | Our Platform |
|---------|---------------|---------|---------|--------------|
| Laravel-specific | ✅ Native | ❌ Generic | ❌ Generic | ✅ Native |
| Server types | ✅ App/Web/DB | ❌ | ❌ | ✅ App/Web/DB/Worker |
| Multi-device | ❌ | ❌ | ❌ | ✅ Intelligent |
| Shared resources | ❌ | Partial | ❌ | ✅ Full |
| Queue workers | ✅ Supervisor | Manual | Manual | ✅ Auto-configured |
| Scheduler | ✅ Cron | Manual | Manual | ✅ Auto-configured |
| Database pooling | ❌ | ❌ | ❌ | ✅ Shared instances |
| Cost | $12/mo + servers | Free (self-host) | Free (self-host) | Free (self-host) |

---

**Version:** 1.0
**Last Updated:** October 2025
**Status:** Design Complete
