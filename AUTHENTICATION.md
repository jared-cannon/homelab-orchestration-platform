# Authentication

## Overview

JWT-based authentication system with bcrypt password hashing and frontend session management.

## Features

- JWT token authentication (24-hour expiry)
- Bcrypt password hashing (cost factor 12)
- First-run admin account creation
- Role-based access control
- Protected routes with automatic redirect
- Cross-tab synchronization
- Session persistence

---

## Setup

### Backend Configuration

Create `.env` in `backend/` directory:

```env
# Enable authentication (REQUIRED for production)
REQUIRE_AUTH=true

# JWT Secret (generate: openssl rand -base64 32)
JWT_SECRET=your-super-secret-jwt-key-here

# Server Configuration
PORT=8080
ENV=development
DB_PATH=./homelab.db

# CORS Configuration
ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000
```

**Generate JWT Secret:**
```bash
openssl rand -base64 32
```

**Start Server:**
```bash
make dev
```

---

## Authentication Flow

### First-Run Setup

1. Navigate to application
2. Automatic redirect to `/login` if unauthenticated
3. Click "Create admin account" → `/setup`
4. Enter username and password (first user = admin)
5. Automatic login and redirect

### Login

1. Navigate to `/login`
2. Enter credentials
3. JWT token issued and stored in localStorage
4. Redirect to devices page

### Logout

- Logout button in navigation
- Token removed from localStorage
- Redirect to login page

---

## API Endpoints

### Public Endpoints

```
POST /api/v1/auth/register   # Create first admin user
POST /api/v1/auth/login      # Login, receive JWT token
```

### Protected Endpoints

```
GET  /api/v1/auth/me                # Current user information
POST /api/v1/auth/change-password   # Change password
GET  /api/v1/devices                # List devices
POST /api/v1/devices                # Create device
```

All device/scanner endpoints require authentication.

---

## Frontend Implementation

### Auth Store

```typescript
import { useAuthStore } from './stores/authStore'

const { user, token, isAuthenticated, isLoading } = useAuthStore()
```

### Auth Hooks

```typescript
import { useLogin, useLogout, useRegister, useCurrentUser } from './hooks/useAuth'

const loginMutation = useLogin()
loginMutation.mutate({ username: 'admin', password: 'password' })

const logout = useLogout()
const { user, isAuthenticated } = useCurrentUser()
```

### Protected Routes

```typescript
import { ProtectedRoute } from './components/ProtectedRoute'

<Route
  path="/devices"
  element={
    <ProtectedRoute>
      <DevicesPage />
    </ProtectedRoute>
  }
/>
```

---

## Security Best Practices

### Production Requirements

1. Enable authentication: `REQUIRE_AUTH=true`
2. Strong JWT secret (minimum 32 characters, secure random)
3. HTTPS only
4. CORS configuration: `ALLOWED_ORIGINS=https://homelab.example.com`
5. Strong passwords (minimum 8 characters enforced)

### Development

Optional: Disable authentication with `REQUIRE_AUTH=false`

**WARNING**: Never disable in production.

---

## Token Management

### Expiry

- 24-hour token lifetime
- Re-login required after expiry
- No automatic refresh

### Storage

- localStorage key: `homelab_auth_token`
- Automatic injection via axios interceptor
- Cleared on logout

### Validation

- Validated on every API request
- Invalid tokens return 401
- Automatic logout on 401

---

## User Management

### Admin Users

- First user automatically becomes admin
- Full access to all features
- User management (planned)

### Regular Users

- Admin creation (planned)
- Limited permissions (planned)
- Device/application access control (planned)

---

## Common Issues

**Authentication middleware DISABLED**
- Set `REQUIRE_AUTH=true` in `.env`

**Invalid or expired token**
- Token expired (>24 hours)
- JWT_SECRET changed
- Clear token: `localStorage.removeItem('homelab_auth_token')`

**CORS errors**
- Add frontend URL to `ALLOWED_ORIGINS`

---

## Reference

- `/backend/internal/middleware/auth.go`
- `/frontend/src/hooks/useAuth.ts`
