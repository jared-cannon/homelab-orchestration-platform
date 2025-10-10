# Authentication Documentation

## Overview

The Homelab Orchestration Platform includes a complete authentication system with JWT-based token authentication, secure password hashing with bcrypt, and a modern frontend auth flow.

## Features

- ✅ **JWT Token Authentication** - Stateless authentication with 24-hour token expiry
- ✅ **Secure Password Storage** - Bcrypt hashing with cost factor 12
- ✅ **First-Run Setup** - Automatic admin account creation on first launch
- ✅ **Role-Based Access** - Admin and regular user roles
- ✅ **Protected Routes** - Automatic redirection for unauthenticated users
- ✅ **Cross-Tab Sync** - Logout in one tab logs out all tabs
- ✅ **Token Persistence** - Automatic session restoration on page reload

## Setup Instructions

### 1. Backend Configuration

Create a `.env` file in the `backend/` directory (use `.env.example` as template):

```bash
# Copy the example file
cp backend/.env.example backend/.env
```

Edit `backend/.env`:

```env
# Enable authentication (REQUIRED for production)
REQUIRE_AUTH=true

# JWT Secret - Generate a secure random key
# Example: openssl rand -base64 32
JWT_SECRET=your-super-secret-jwt-key-here

# Server Configuration
PORT=8080
ENV=development
DB_PATH=./homelab.db

# CORS Configuration
ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000
```

### 2. Generate Secure JWT Secret

For production, generate a secure random JWT secret:

```bash
# On macOS/Linux
openssl rand -base64 32

# Or use a password generator
# Minimum 32 characters recommended
```

### 3. Start the Server

```bash
# From project root
make dev

# Or manually
cd backend
go run cmd/server/main.go
```

## Authentication Flow

### First-Run Setup

1. **Visit the application** - Navigate to `http://localhost:5173`
2. **Automatic redirect** - If not authenticated, redirected to `/login`
3. **Setup link** - Click "Create admin account" to go to `/setup`
4. **Create admin** - Enter username and password (first user becomes admin)
5. **Automatic login** - After setup, automatically logged in and redirected to app

### Subsequent Logins

1. **Visit `/login`** - Enter username and password
2. **Token issued** - JWT token issued and stored in localStorage
3. **Redirect** - Automatically redirected to the devices page
4. **Session persistence** - Token validated on page reload

### Logout

- Click "Logout" button in top navigation bar
- Token removed from localStorage
- Redirected to login page
- Cross-tab logout (all tabs logged out simultaneously)

## API Endpoints

### Public Endpoints (No Authentication Required)

- `POST /api/v1/auth/register` - Create first admin user
- `POST /api/v1/auth/login` - Login and receive JWT token

### Protected Endpoints (Authentication Required)

- `GET /api/v1/auth/me` - Get current user information
- `POST /api/v1/auth/change-password` - Change user password
- `GET /api/v1/devices` - List all devices
- `POST /api/v1/devices` - Create new device
- ... (all device/scanner endpoints)

## Frontend Implementation

### Auth Store (Zustand)

Global state management for authentication:

```typescript
import { useAuthStore } from './stores/authStore'

// In a component
const { user, token, isAuthenticated, isLoading } = useAuthStore()
```

### Auth Hooks

Convenient React hooks for auth operations:

```typescript
import { useLogin, useLogout, useRegister, useCurrentUser } from './hooks/useAuth'

// Login
const loginMutation = useLogin()
loginMutation.mutate({ username: 'admin', password: 'password' })

// Logout
const logout = useLogout()
logout()

// Get current user
const { user, isAuthenticated } = useCurrentUser()
```

### Protected Routes

Wrap routes that require authentication:

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

## Security Best Practices

### Production Deployment

1. **Enable Authentication**
   ```env
   REQUIRE_AUTH=true
   ```

2. **Strong JWT Secret**
   - Minimum 32 characters
   - Generated from secure random source
   - Never commit to version control

3. **HTTPS Only**
   - Always use HTTPS in production
   - HTTP is insecure for authentication

4. **CORS Configuration**
   ```env
   # Only allow your frontend domain
   ALLOWED_ORIGINS=https://homelab.example.com
   ```

5. **Secure Password Requirements**
   - Minimum 8 characters (enforced by backend)
   - Consider enforcing complexity requirements
   - Use strong passwords for admin accounts

### Development

For development, you can disable authentication:

```env
REQUIRE_AUTH=false
```

**⚠️ WARNING**: Never disable authentication in production!

## Token Management

### Token Expiry

- Tokens expire after 24 hours
- Users must re-login after expiry
- No automatic token refresh (by design)

### Token Storage

- Stored in localStorage with key `homelab_auth_token`
- Automatically injected in API requests via axios interceptor
- Cleared on logout

### Token Validation

- Validated on every API request
- Invalid tokens return 401 Unauthorized
- Frontend automatically logs out on 401

## User Management

### Admin Users

- First user is automatically admin
- Admins have full access to all features
- Can manage other users (future feature)

### Regular Users

- Created by admins (future feature)
- Limited permissions (future feature)
- Can access own devices and applications

## Troubleshooting

### "Authentication middleware DISABLED" Warning

**Problem**: Server logs show auth is disabled

**Solution**: Set `REQUIRE_AUTH=true` in `.env` file

### "Invalid or expired token" Error

**Problem**: Token validation fails

**Possible causes**:
1. Token expired (>24 hours old) - Re-login
2. JWT_SECRET changed - Clear localStorage and re-login
3. Invalid token format - Clear localStorage and re-login

**Solution**:
```javascript
// Open browser console
localStorage.removeItem('homelab_auth_token')
// Then reload page and login again
```

### "Username already exists" on Setup

**Problem**: Trying to create admin when users exist

**Solution**: Navigate to `/login` instead of `/setup`

### CORS Errors

**Problem**: Frontend can't connect to backend

**Solution**: Add frontend URL to `ALLOWED_ORIGINS`:
```env
ALLOWED_ORIGINS=http://localhost:5173,http://192.168.1.100:5173
```

## Testing

### Test the Auth Flow

1. **Clean database** (optional, for testing):
   ```bash
   rm backend/homelab.db
   ```

2. **Start servers**:
   ```bash
   make dev
   ```

3. **Test setup** - Visit `http://localhost:5173/setup`
   - Create admin account
   - Verify redirect to devices page
   - Check top nav shows username

4. **Test logout**:
   - Click logout button
   - Verify redirect to login
   - Verify can't access `/` without auth

5. **Test login**:
   - Enter credentials on login page
   - Verify redirect to devices
   - Verify token persists on reload

6. **Test protected routes**:
   - Logout
   - Try to access `http://localhost:5173/`
   - Verify automatic redirect to `/login`

## Future Enhancements

- [ ] Refresh tokens for long-lived sessions
- [ ] Password reset via email
- [ ] Two-factor authentication (2FA)
- [ ] OAuth integration (Google, GitHub)
- [ ] User management UI for admins
- [ ] Role-based permissions system
- [ ] API rate limiting
- [ ] Audit logging for auth events

## Support

For issues or questions about authentication:
1. Check this documentation
2. Review `/backend/internal/middleware/auth.go`
3. Review `/frontend/src/hooks/useAuth.ts`
4. Open an issue on GitHub
