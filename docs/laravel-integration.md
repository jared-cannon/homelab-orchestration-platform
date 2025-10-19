# Laravel Integration Guide

## Overview

First-class Laravel support for homelab deployment with zero-configuration setup. Deploy Laravel applications with the same ease as Laravel Forge, but on your own infrastructure.

## Key Features

- One-click Laravel app server provisioning
- Automatic environment detection
- Queue worker auto-configuration (Supervisor + Horizon)
- Scheduler auto-setup (Cron)
- Database auto-provisioning
- Redis caching
- Asset building (Vite/Mix)
- Zero-downtime deployments
- Environment management (production, staging, development)

## Quick Start

### 1. Provision Laravel App Server

**UI Flow:**
```
Dashboard → Provision → App Server → Laravel App Server
    ↓
Configure:
  - PHP Version: 8.3
  - Database: PostgreSQL 15
  - Redis: Yes
  - Supervisor: Yes (queue workers)
    ↓
Deploy to: homelab-server-02 (auto-selected)
    ↓
Wait 3-5 minutes → Server ready
```

**What Gets Installed:**
- Nginx (configured for Laravel)
- PHP 8.3 + required extensions
- Composer 2.x
- Node.js 20.x + npm
- PostgreSQL 15
- Redis 7.x
- Supervisor (for queues)
- Cron (for scheduler)

### 2. Deploy Laravel Application

**UI Flow:**
```
Infrastructure → Deploy Laravel App
    ↓
Enter:
  - Git Repository: https://github.com/user/my-laravel-app
  - Branch: main
  - Server: laravel-production
  - Domain: app.homelab.local
    ↓
Auto-detected:
  - PHP Version: 8.3 (from composer.json)
  - Database: PostgreSQL (from .env.example)
  - Queue Driver: redis (from config/queue.php)
    ↓
Deploy → Wait 2-3 minutes → App live at https://app.homelab.local
```

## Environment Detection

The system automatically analyzes your Laravel application to configure the correct environment:

### composer.json Analysis

```json
{
  "require": {
    "php": "^8.2",
    "laravel/framework": "^11.0",
    "laravel/horizon": "^5.20"
  }
}
```

**Detected Configuration:**
- PHP Version: 8.2+ (will provision 8.3)
- Laravel Version: 11.x
- Queue Dashboard: Horizon (will auto-configure)

### .env.example Analysis

```env
DB_CONNECTION=pgsql
DB_HOST=127.0.0.1
DB_PORT=5432
DB_DATABASE=laravel

CACHE_DRIVER=redis
SESSION_DRIVER=redis
QUEUE_CONNECTION=redis

REDIS_HOST=127.0.0.1
REDIS_PASSWORD=null
REDIS_PORT=6379
```

**Detected Requirements:**
- Database: PostgreSQL
- Cache: Redis
- Session: Redis
- Queue: Redis

**Auto-Provisioning:**
1. Create PostgreSQL database `myapp_production`
2. Create database user with secure password
3. Deploy/use shared Redis instance
4. Generate `.env` file with correct credentials

### config/queue.php Analysis

```php
'connections' => [
    'redis' => [
        'driver' => 'redis',
        'connection' => 'default',
        'queue' => env('REDIS_QUEUE', 'default'),
        'retry_after' => 90,
    ],
],
```

**Detected Queue Configuration:**
- Driver: Redis
- Will configure Supervisor workers

### vite.config.js / webpack.mix.js Analysis

```js
import { defineConfig } from 'vite';
import laravel from 'laravel-vite-plugin';
import vue from '@vitejs/plugin-vue';

export default defineConfig({
    plugins: [
        laravel(['resources/css/app.css', 'resources/js/app.js']),
        vue(),
    ],
});
```

**Detected Build Configuration:**
- Build Tool: Vite
- Frontend Framework: Vue.js
- Will run: `npm install && npm run build`

## Deployment Process

### Standard Deployment Flow

```
┌─────────────────────────────────────────────────┐
│ 1. Clone Repository                             │
│    git clone https://github.com/user/app       │
│    cd app                                       │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 2. Analyze Application                          │
│    • Detect PHP version (composer.json)        │
│    • Detect database requirements              │
│    • Detect queue configuration                │
│    • Detect build tool (Vite/Mix)              │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 3. Provision Infrastructure                     │
│    • Create database (if needed)               │
│    • Configure Redis (if needed)               │
│    • Generate .env file                        │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 4. Install Dependencies                         │
│    composer install --no-dev --optimize-autoloader │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 5. Build Assets                                 │
│    npm ci                                       │
│    npm run build                                │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 6. Run Migrations                               │
│    php artisan migrate --force                 │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 7. Optimize Application                         │
│    php artisan config:cache                    │
│    php artisan route:cache                     │
│    php artisan view:cache                      │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 8. Configure Queue Workers                      │
│    • Generate Supervisor config                │
│    • Start workers                             │
│    • Configure Horizon (if detected)           │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 9. Configure Scheduler                          │
│    • Add cron entry                            │
│    • Verify schedule:run                       │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 10. Configure Nginx                             │
│     • Generate vhost configuration             │
│     • Test config (nginx -t)                   │
│     • Reload Nginx                             │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 11. Health Check                                │
│     curl https://app.homelab.local/health      │
└─────────────────────────────────────────────────┘
                    ↓
              ✅ DEPLOYED
```

### Real-Time Progress Updates

```
┌────────────────────────────────────────────────────┐
│  Deploying Laravel Application...                 │
│                                                    │
│  ✅ Cloned repository (1/11) - 8 seconds          │
│  ✅ Analyzed application (2/11) - 2 seconds       │
│     Detected: Laravel 11, PHP 8.3, PostgreSQL    │
│  ✅ Provisioned database (3/11) - 15 seconds      │
│     Created: myapp_production                     │
│  ✅ Installed Composer dependencies (4/11) - 42s  │
│     Installed 87 packages                         │
│  ✅ Installed NPM dependencies (5/11) - 38s       │
│  ✅ Built assets with Vite (6/11) - 25 seconds    │
│  ✅ Ran migrations (7/11) - 5 seconds             │
│     Migrated: 12 migrations                       │
│  ✅ Optimized application (8/11) - 3 seconds      │
│  ✅ Configured queue workers (9/11) - 8 seconds   │
│     Started 4 workers                             │
│  ✅ Configured scheduler (10/11) - 2 seconds      │
│  ⏳ Configuring Nginx (11/11)...                  │
│                                                    │
│  Progress: ███████████████████░ 95%               │
└────────────────────────────────────────────────────┘
```

## Queue Worker Configuration

### Automatic Supervisor Setup

For standard queue workers:

**Generated Supervisor Config:**
```ini
[program:myapp-worker]
process_name=%(program_name)s_%(process_num)02d
command=php /var/www/myapp/artisan queue:work redis --sleep=3 --tries=3 --max-time=3600
autostart=true
autorestart=true
stopasgroup=true
killasgroup=true
user=www-data
numprocs=4
redirect_stderr=true
stdout_logfile=/var/www/myapp/storage/logs/worker.log
stopwaitsecs=3600
```

**Worker Management UI:**
```
┌────────────────────────────────────────────────────┐
│  Queue Workers: myapp-production                  │
│                                                    │
│  Status: ✅ Running (4 workers)                   │
│                                                    │
│  Workers:                                          │
│  • myapp-worker_00: ✅ Running (5 jobs processed) │
│  • myapp-worker_01: ✅ Running (8 jobs processed) │
│  • myapp-worker_02: ✅ Running (3 jobs processed) │
│  • myapp-worker_03: ✅ Running (6 jobs processed) │
│                                                    │
│  [Restart Workers] [View Logs] [Configure]        │
└────────────────────────────────────────────────────┘
```

### Laravel Horizon Support

If Horizon is detected in `composer.json`, automatically configure:

**Installation:**
```bash
cd /var/www/myapp
composer require laravel/horizon
php artisan horizon:install
php artisan migrate
```

**Supervisor Config for Horizon:**
```ini
[program:myapp-horizon]
process_name=%(program_name)s
command=php /var/www/myapp/artisan horizon
autostart=true
autorestart=true
user=www-data
redirect_stderr=true
stdout_logfile=/var/www/myapp/storage/logs/horizon.log
stopwaitsecs=3600
```

**Horizon Dashboard Access:**
```
App deployed with Horizon ✅

Dashboard: https://app.homelab.local/horizon
```

## Scheduler Configuration

### Automatic Cron Setup

**Cron Entry:**
```bash
* * * * * cd /var/www/myapp && php artisan schedule:run >> /dev/null 2>&1
```

**Verification:**
```bash
# After deployment, verify scheduler is working
php artisan schedule:list

┌────────────────────────────────────────────────┬──────────┐
│ Command                                        │ Interval │
├────────────────────────────────────────────────┼──────────┤
│ php artisan queue:prune-batches --hours=48     │ Daily    │
│ php artisan backup:clean                       │ Daily    │
│ php artisan backup:run                         │ Daily    │
└────────────────────────────────────────────────┴──────────┘
```

## Environment Variables

### Auto-Generated .env

Based on detected requirements and provisioned infrastructure:

```env
# Generated by Homelab Orchestration Platform
# Application: myapp-production
# Deployed: 2025-10-15 14:30:00

APP_NAME="My Laravel App"
APP_ENV=production
APP_KEY=base64:GENERATED_KEY_HERE
APP_DEBUG=false
APP_URL=https://app.homelab.local

LOG_CHANNEL=stack
LOG_DEPRECATIONS_CHANNEL=null
LOG_LEVEL=error

# Database (Auto-provisioned PostgreSQL)
DB_CONNECTION=pgsql
DB_HOST=192.168.1.105
DB_PORT=5432
DB_DATABASE=myapp_production
DB_USERNAME=myapp_user
DB_PASSWORD=GENERATED_SECURE_PASSWORD

# Cache & Session (Shared Redis instance)
CACHE_DRIVER=redis
SESSION_DRIVER=redis
SESSION_LIFETIME=120

# Queue (Shared Redis instance)
QUEUE_CONNECTION=redis

# Redis Configuration
REDIS_HOST=192.168.1.105
REDIS_PASSWORD=null
REDIS_PORT=6379
REDIS_DB=0
REDIS_CACHE_DB=1

# Broadcasting (if needed)
BROADCAST_DRIVER=log

# Mail (configure manually)
MAIL_MAILER=smtp
MAIL_HOST=mailhog
MAIL_PORT=1025
MAIL_USERNAME=null
MAIL_PASSWORD=null
MAIL_ENCRYPTION=null
MAIL_FROM_ADDRESS="hello@example.com"
MAIL_FROM_NAME="${APP_NAME}"
```

### Environment Management

**UI for editing .env:**
```
┌────────────────────────────────────────────────────┐
│  Environment Variables: myapp-production          │
│                                                    │
│  ┌────────────────────┬──────────────────────────┐│
│  │ Key                │ Value                    ││
│  ├────────────────────┼──────────────────────────┤│
│  │ APP_NAME           │ My Laravel App        [E]││
│  │ APP_ENV            │ production            [E]││
│  │ APP_DEBUG          │ false                 [E]││
│  │ APP_URL            │ https://app.local     [E]││
│  │ DB_CONNECTION      │ pgsql (auto)             ││
│  │ DB_HOST            │ 192.168.1.105 (auto)     ││
│  │ DB_DATABASE        │ myapp_production (auto)  ││
│  │ MAIL_MAILER        │ smtp                  [E]││
│  │ MAIL_HOST          │ (not set)             [+]││
│  └────────────────────┴──────────────────────────┘│
│                                                    │
│  [E] = Edit  [+] = Add Variable                   │
│                                                    │
│  ⚠️ Changes require redeployment to take effect   │
│                                                    │
│  [Add Variable] [Import .env] [Redeploy]          │
└────────────────────────────────────────────────────┘
```

## Nginx Configuration

### Auto-Generated Nginx vhost

```nginx
server {
    listen 80;
    listen [::]:80;
    server_name app.homelab.local;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name app.homelab.local;

    # SSL (via Traefik or Let's Encrypt)
    ssl_certificate /etc/ssl/certs/app.homelab.local.crt;
    ssl_certificate_key /etc/ssl/private/app.homelab.local.key;

    root /var/www/myapp/public;
    index index.php;

    # Laravel-optimized configuration
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    # PHP-FPM
    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/var/run/php/php8.3-fpm.sock;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;

        # Laravel-specific
        fastcgi_buffers 16 16k;
        fastcgi_buffer_size 32k;
    }

    # Deny access to hidden files
    location ~ /\.(?!well-known).* {
        deny all;
    }

    # Deny access to sensitive files
    location ~ ^/(composer\.json|composer\.lock|package\.json|package-lock\.json|\.env.*) {
        deny all;
    }

    # Static assets
    location ~* \.(jpg|jpeg|png|gif|ico|css|js|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

## Multiple Environments

### Environment Types

**Production:**
- Domain: app.homelab.local
- Database: myapp_production
- Debug: false
- Queues: 4 workers
- Caching: Full (config, routes, views)

**Staging:**
- Domain: staging.homelab.local
- Database: myapp_staging
- Debug: false
- Queues: 2 workers
- Caching: Partial

**Development:**
- Domain: dev.homelab.local
- Database: myapp_development
- Debug: true
- Queues: 1 worker
- Caching: None (for debugging)

### UI for Environment Management

```
┌────────────────────────────────────────────────────┐
│  Environments: My Laravel App                     │
│                                                    │
│  ┌────────────────────────────────────────────┐   │
│  │ Production                     ✅ Running  │   │
│  │ app.homelab.local                          │   │
│  │ Server: laravel-production                 │   │
│  │ Last Deploy: 2 hours ago                   │   │
│  │ [Deploy] [Rollback] [Logs] [Shell]         │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  ┌────────────────────────────────────────────┐   │
│  │ Staging                        ✅ Running  │   │
│  │ staging.homelab.local                      │   │
│  │ Server: laravel-staging                    │   │
│  │ Last Deploy: 1 day ago                     │   │
│  │ [Deploy] [Rollback] [Logs] [Shell]         │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  ┌────────────────────────────────────────────┐   │
│  │ Development                    ⚪ Not Setup │   │
│  │ [Create Development Environment]           │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
└────────────────────────────────────────────────────┘
```

## Deployment Hooks

### Pre-Deployment Hooks

Execute custom commands before deployment:

```yaml
# .homelab/deploy.yml in your Laravel repo

pre_deploy:
  - command: php artisan down
    description: "Put application in maintenance mode"

  - command: php artisan backup:run
    description: "Backup database before deploy"

deploy:
  composer_install: true
  npm_install: true
  build_assets: true
  run_migrations: true
  clear_cache: true

post_deploy:
  - command: php artisan up
    description: "Bring application back online"

  - command: php artisan queue:restart
    description: "Restart queue workers"

  - command: php artisan horizon:terminate
    description: "Gracefully terminate Horizon"
    if: horizon_detected

  - command: php artisan cache:clear
    description: "Clear application cache"

  - command: curl https://healthchecks.io/ping/YOUR_UUID
    description: "Notify monitoring service"
```

## Rollback Support

### Automatic Rollback

If deployment fails (e.g., migrations fail, health check fails), automatically rollback:

```
┌────────────────────────────────────────────────────┐
│  ⚠️ Deployment Failed                              │
│                                                    │
│  Step: Run migrations (7/11)                      │
│  Error: SQLSTATE[42S01]: Base table exists        │
│                                                    │
│  Rolling back to previous version...              │
│                                                    │
│  ✅ Restored previous code (git checkout)         │
│  ✅ Restored database (from backup)               │
│  ✅ Restarted services                            │
│                                                    │
│  Application is back online at previous version.  │
│                                                    │
│  [View Error Log] [Retry Deployment] [Close]      │
└────────────────────────────────────────────────────┘
```

### Manual Rollback

```
┌────────────────────────────────────────────────────┐
│  Deployment History: myapp-production             │
│                                                    │
│  ┌────────────────────────────────────────────┐   │
│  │ v1.5.2 - CURRENT                           │   │
│  │ Deployed: 2 hours ago                      │   │
│  │ Commit: abc1234 "Add new feature"          │   │
│  │ Status: ✅ Running                         │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  ┌────────────────────────────────────────────┐   │
│  │ v1.5.1                                     │   │
│  │ Deployed: 1 day ago                        │   │
│  │ Commit: def5678 "Bug fix"                  │   │
│  │ [Rollback to this version]                 │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  ┌────────────────────────────────────────────┐   │
│  │ v1.5.0                                     │   │
│  │ Deployed: 3 days ago                       │   │
│  │ Commit: ghi9012 "Major update"             │   │
│  │ [Rollback to this version]                 │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
└────────────────────────────────────────────────────┘
```

## Logging & Debugging

### Log Viewer

Real-time log streaming:

```
┌────────────────────────────────────────────────────┐
│  Logs: myapp-production                           │
│                                                    │
│  Source: ┌──────────────────────────────────┐     │
│          │ Application (storage/logs)    ▼ │     │
│          └──────────────────────────────────┘     │
│  Options: Workers, Horizon, Nginx, PHP-FPM       │
│                                                    │
│  ┌────────────────────────────────────────────┐   │
│  │ [2025-10-15 14:45:23] production.INFO:    │   │
│  │ User logged in {"user_id":42}             │   │
│  │                                            │   │
│  │ [2025-10-15 14:45:25] production.ERROR:   │   │
│  │ SQLSTATE[23000]: Integrity constraint     │   │
│  │ violation...                               │   │
│  │                                            │   │
│  │ [2025-10-15 14:45:30] production.INFO:    │   │
│  │ Cache cleared successfully                 │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  [Pause] [Download] [Clear] [Filter]              │
└────────────────────────────────────────────────────┘
```

### SSH Shell Access

```
┌────────────────────────────────────────────────────┐
│  Shell: myapp-production @ laravel-production     │
│                                                    │
│  www-data@homelab-server-02:/var/www/myapp$       │
│  _                                                 │
│                                                    │
│  Quick Commands:                                   │
│  [php artisan tinker]                             │
│  [php artisan queue:work]                         │
│  [php artisan migrate]                            │
│  [composer install]                               │
│  [npm run dev]                                    │
│                                                    │
└────────────────────────────────────────────────────┘
```

## Performance Optimization

### OPcache Configuration

Auto-configured for production:

```ini
; /etc/php/8.3/fpm/conf.d/99-opcache.ini
opcache.enable=1
opcache.memory_consumption=256
opcache.interned_strings_buffer=16
opcache.max_accelerated_files=10000
opcache.revalidate_freq=0
opcache.validate_timestamps=0  ; Disable in production
opcache.save_comments=1
opcache.fast_shutdown=1
```

### PHP-FPM Tuning

Based on available RAM:

```ini
; /etc/php/8.3/fpm/pool.d/laravel.conf
[laravel]
user = www-data
group = www-data

listen = /var/run/php/php8.3-fpm-laravel.sock
listen.owner = www-data
listen.group = www-data

pm = dynamic
pm.max_children = 20        ; Adjusted based on RAM
pm.start_servers = 5
pm.min_spare_servers = 2
pm.max_spare_servers = 10
pm.max_requests = 500
```

---

**Version:** 1.0
**Last Updated:** October 2025
**Status:** Design Complete
