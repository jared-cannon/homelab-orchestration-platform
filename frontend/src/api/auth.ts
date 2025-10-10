import type { User } from '../stores/authStore'

const API_BASE_URL = '/api/v1'

// Request types
export interface LoginRequest {
  username: string
  password: string
}

export interface RegisterRequest {
  username: string
  password: string
  email?: string
}

export interface ChangePasswordRequest {
  old_password: string
  new_password: string
}

// Response types
export interface AuthResponse {
  token: string
  username: string
  is_admin: boolean
  user: User
}

export interface ChangePasswordResponse {
  message: string
}

class AuthAPIClient {
  private async request<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({}))
      throw new Error(error.error || `HTTP ${response.status}`)
    }

    if (response.status === 204) {
      return {} as T
    }

    return response.json()
  }

  /**
   * Login with username and password
   * @param username - User's username
   * @param password - User's password
   * @returns AuthResponse with token and user information
   */
  async login(username: string, password: string): Promise<AuthResponse> {
    return this.request<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    })
  }

  /**
   * Register a new user (first user becomes admin)
   * @param username - Desired username
   * @param password - User's password
   * @param email - Optional email address
   * @returns AuthResponse with token and user information
   */
  async register(
    username: string,
    password: string,
    email?: string
  ): Promise<AuthResponse> {
    return this.request<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, password, email }),
    })
  }

  /**
   * Get current user information (validates token)
   * @param token - JWT token
   * @returns User object
   */
  async getCurrentUser(token: string): Promise<User> {
    return this.request<User>('/auth/me', {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })
  }

  /**
   * Change user's password
   * @param token - JWT token
   * @param oldPassword - Current password
   * @param newPassword - New password
   * @returns Success message
   */
  async changePassword(
    token: string,
    oldPassword: string,
    newPassword: string
  ): Promise<ChangePasswordResponse> {
    return this.request<ChangePasswordResponse>('/auth/change-password', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        old_password: oldPassword,
        new_password: newPassword,
      }),
    })
  }

  /**
   * Check if any users exist in the system
   * This is used to determine if we should show setup page or login page
   * We do this by attempting to get current user with no token
   * If there are no users, register endpoint will be available
   */
  async checkSetupRequired(): Promise<boolean> {
    try {
      // Try to access a protected endpoint without auth
      // If no users exist, we need setup
      const response = await fetch(`${API_BASE_URL}/auth/me`)
      // If we get 401, there might be users (auth is enforced)
      // If we get other errors, handle accordingly
      return response.status === 401
    } catch {
      // If request fails completely, assume setup is needed
      return true
    }
  }
}

export const authAPI = new AuthAPIClient()
