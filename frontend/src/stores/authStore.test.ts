import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore, type User } from './authStore'

const mockUser: User = {
  id: '123',
  username: 'testuser',
  email: 'test@example.com',
  is_admin: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
}

describe('authStore', () => {
  beforeEach(() => {
    // Clear store state before each test
    useAuthStore.getState().logout()
    localStorage.clear()
  })

  it('should have initial state', () => {
    const state = useAuthStore.getState()
    expect(state.user).toBeNull()
    expect(state.token).toBeNull()
    expect(state.isAuthenticated).toBe(false)
    expect(state.isLoading).toBe(true)
  })

  describe('login', () => {
    it('should set token and user on login', () => {
      const token = 'test-token'
      useAuthStore.getState().login(token, mockUser)

      const state = useAuthStore.getState()
      expect(state.token).toBe(token)
      expect(state.user).toEqual(mockUser)
      expect(state.isAuthenticated).toBe(true)
      expect(state.isLoading).toBe(false)
    })

    it('should persist token to localStorage', () => {
      const token = 'test-token'
      useAuthStore.getState().login(token, mockUser)

      expect(localStorage.getItem('homelab_auth_token')).toBe(token)
    })
  })

  describe('logout', () => {
    it('should clear token and user on logout', () => {
      // First login
      useAuthStore.getState().login('test-token', mockUser)

      // Then logout
      useAuthStore.getState().logout()

      const state = useAuthStore.getState()
      expect(state.token).toBeNull()
      expect(state.user).toBeNull()
      expect(state.isAuthenticated).toBe(false)
      expect(state.isLoading).toBe(false)
    })

    it('should remove token from localStorage', () => {
      // First login
      useAuthStore.getState().login('test-token', mockUser)
      expect(localStorage.getItem('homelab_auth_token')).toBe('test-token')

      // Then logout
      useAuthStore.getState().logout()
      expect(localStorage.getItem('homelab_auth_token')).toBeNull()
    })
  })

  describe('setUser', () => {
    it('should set user and mark as authenticated', () => {
      useAuthStore.getState().setUser(mockUser)

      const state = useAuthStore.getState()
      expect(state.user).toEqual(mockUser)
      expect(state.isAuthenticated).toBe(true)
    })

    it('should clear authentication when user is null', () => {
      // First set a user
      useAuthStore.getState().setUser(mockUser)
      expect(useAuthStore.getState().isAuthenticated).toBe(true)

      // Then clear
      useAuthStore.getState().setUser(null)
      expect(useAuthStore.getState().user).toBeNull()
      expect(useAuthStore.getState().isAuthenticated).toBe(false)
    })
  })

  describe('setToken', () => {
    it('should set token and persist to localStorage', () => {
      const token = 'new-token'
      useAuthStore.getState().setToken(token)

      expect(useAuthStore.getState().token).toBe(token)
      expect(localStorage.getItem('homelab_auth_token')).toBe(token)
    })

    it('should remove token from localStorage when null', () => {
      // First set a token
      useAuthStore.getState().setToken('test-token')
      expect(localStorage.getItem('homelab_auth_token')).toBe('test-token')

      // Then clear
      useAuthStore.getState().setToken(null)
      expect(useAuthStore.getState().token).toBeNull()
      expect(localStorage.getItem('homelab_auth_token')).toBeNull()
    })
  })

  describe('setLoading', () => {
    it('should update loading state', () => {
      useAuthStore.getState().setLoading(true)
      expect(useAuthStore.getState().isLoading).toBe(true)

      useAuthStore.getState().setLoading(false)
      expect(useAuthStore.getState().isLoading).toBe(false)
    })
  })

  describe('initializeAuth', () => {
    it('should load token from localStorage if exists', () => {
      localStorage.setItem('homelab_auth_token', 'stored-token')

      useAuthStore.getState().initializeAuth()

      const state = useAuthStore.getState()
      expect(state.token).toBe('stored-token')
      expect(state.isLoading).toBe(true) // Loading while validating
    })

    it('should set loading to false if no token exists', () => {
      useAuthStore.getState().initializeAuth()

      const state = useAuthStore.getState()
      expect(state.token).toBeNull()
      expect(state.isLoading).toBe(false)
    })
  })
})
