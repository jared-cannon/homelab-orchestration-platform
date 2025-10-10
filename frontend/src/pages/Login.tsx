import { useState } from 'react'
import { Link } from 'react-router-dom'
import { toast } from 'sonner'
import { useLogin } from '../hooks/useAuth'

export function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const loginMutation = useLogin()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!username || !password) {
      toast.error('Please fill in all fields')
      return
    }

    try {
      await loginMutation.mutateAsync({ username, password })
      toast.success('Login successful')
    } catch (error) {
      toast.error('Login failed', {
        description: (error as Error).message || 'Invalid username or password',
      })
    }
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted flex items-center justify-center px-4">
      <div className="w-full max-w-md">
        {/* Logo/Header */}
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold mb-2">Homelab Orchestration</h1>
          <p className="text-muted-foreground">Sign in to your account</p>
        </div>

        {/* Login Form Card */}
        <div className="bg-card border border-border rounded-lg shadow-lg p-8">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Username Field */}
            <div>
              <label
                htmlFor="username"
                className="block text-sm font-medium mb-2"
              >
                Username
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full px-4 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                placeholder="Enter your username"
                autoComplete="username"
                disabled={loginMutation.isPending}
              />
            </div>

            {/* Password Field */}
            <div>
              <label
                htmlFor="password"
                className="block text-sm font-medium mb-2"
              >
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full px-4 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
                placeholder="Enter your password"
                autoComplete="current-password"
                disabled={loginMutation.isPending}
              />
            </div>

            {/* Submit Button */}
            <button
              type="submit"
              disabled={loginMutation.isPending}
              className="w-full bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed font-medium py-2 px-4 rounded-md transition-colors"
            >
              {loginMutation.isPending ? 'Signing in...' : 'Sign in'}
            </button>
          </form>

          {/* Setup Link */}
          <div className="mt-6 text-center">
            <p className="text-sm text-muted-foreground">
              First time setup?{' '}
              <Link
                to="/setup"
                className="text-primary hover:underline font-medium"
              >
                Create admin account
              </Link>
            </p>
          </div>
        </div>

        {/* Footer */}
        <div className="text-center mt-8 text-sm text-muted-foreground">
          <p>Homelab Orchestration Platform v0.1.0</p>
        </div>
      </div>
    </div>
  )
}
