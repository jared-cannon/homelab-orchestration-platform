import { create } from 'zustand'

const TOKEN_STORAGE_KEY = 'homelab_auth_token'

interface User {
  id: string
  username: string
  email?: string
  is_admin: boolean
  created_at: string
  updated_at: string
}

interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  isLoading: boolean
}

interface AuthActions {
  login: (token: string, user: User) => void
  logout: () => void
  setUser: (user: User | null) => void
  setToken: (token: string | null) => void
  setLoading: (isLoading: boolean) => void
  initializeAuth: () => void
}

type AuthStore = AuthState & AuthActions

export const useAuthStore = create<AuthStore>((set) => ({
  // Initial state
  user: null,
  token: null,
  isAuthenticated: false,
  isLoading: true,

  // Actions
  login: (token: string, user: User) => {
    localStorage.setItem(TOKEN_STORAGE_KEY, token)
    set({
      token,
      user,
      isAuthenticated: true,
      isLoading: false,
    })
  },

  logout: () => {
    localStorage.removeItem(TOKEN_STORAGE_KEY)
    set({
      token: null,
      user: null,
      isAuthenticated: false,
      isLoading: false,
    })
  },

  setUser: (user: User | null) => {
    set({ user, isAuthenticated: user !== null })
  },

  setToken: (token: string | null) => {
    if (token) {
      localStorage.setItem(TOKEN_STORAGE_KEY, token)
    } else {
      localStorage.removeItem(TOKEN_STORAGE_KEY)
    }
    set({ token })
  },

  setLoading: (isLoading: boolean) => {
    set({ isLoading })
  },

  initializeAuth: () => {
    const token = localStorage.getItem(TOKEN_STORAGE_KEY)
    if (token) {
      set({ token, isLoading: true })
      // Token validation will be handled by AuthProvider
    } else {
      set({ isLoading: false })
    }
  },
}))

// Sync auth state across tabs
if (typeof window !== 'undefined') {
  window.addEventListener('storage', (event) => {
    if (event.key === TOKEN_STORAGE_KEY) {
      if (event.newValue) {
        // Token was set in another tab
        useAuthStore.getState().setToken(event.newValue)
      } else {
        // Token was removed in another tab
        useAuthStore.getState().logout()
      }
    }
  })
}

// Export types
export type { User, AuthState, AuthActions, AuthStore }
