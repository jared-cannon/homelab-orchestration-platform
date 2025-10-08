import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '../test/utils'
import userEvent from '@testing-library/user-event'
import { AddDeviceDialog } from './AddDeviceDialog'
import * as hooks from '../api/hooks'

// Mock the toast notifications
vi.mock('sonner', () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

describe('AddDeviceDialog', () => {
  const mockMutateAsync = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()

    // Mock the hooks with minimal implementation
    vi.spyOn(hooks, 'useCreateDevice').mockReturnValue({
      mutateAsync: mockMutateAsync,
      isPending: false,
    } as any)

    vi.spyOn(hooks, 'useTestConnectionBeforeCreate').mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: false,
    } as any)
  })

  it('renders Add Device button', () => {
    render(<AddDeviceDialog />)
    expect(screen.getByRole('button', { name: /Add Device/i })).toBeInTheDocument()
  })

  it('opens dialog when button is clicked', async () => {
    const user = userEvent.setup()
    render(<AddDeviceDialog />)

    const button = screen.getByRole('button', { name: /Add Device/i })
    await user.click(button)

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText(/Add New Device/i)).toBeInTheDocument()
  })

  it('displays all required form fields', async () => {
    const user = userEvent.setup()
    render(<AddDeviceDialog />)

    await user.click(screen.getByRole('button', { name: /Add Device/i }))

    // Check for all form fields
    expect(screen.getByLabelText(/Device Name/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/Device Type/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/IP Address/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/MAC Address/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/Login Method/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/Username/i)).toBeInTheDocument()
  })

  it('shows password field by default', async () => {
    const user = userEvent.setup()
    render(<AddDeviceDialog />)

    await user.click(screen.getByRole('button', { name: /Add Device/i }))

    expect(screen.getByLabelText(/^Password$/i)).toBeInTheDocument()
    expect(screen.queryByLabelText(/Security Key/i)).not.toBeInTheDocument()
  })

  it.skip('switches to SSH key fields when selected', async () => {
    const user = userEvent.setup()
    render(<AddDeviceDialog />)

    await user.click(screen.getByRole('button', { name: /Add Device/i }))

    // Click on the Login Method select trigger
    const loginMethodTrigger = screen.getByRole('combobox', { name: /Login Method/i })
    await user.click(loginMethodTrigger)

    // Select SSH Key option by clicking the text
    const sshKeyOption = await screen.findByText('Security Key')
    await user.click(sshKeyOption)

    // Verify SSH key fields appear
    await waitFor(() => {
      expect(screen.getByLabelText(/^Security Key$/i)).toBeInTheDocument()
    })
    expect(screen.getByLabelText(/Key Password/i)).toBeInTheDocument()
    expect(screen.queryByLabelText(/^Password$/i)).not.toBeInTheDocument()
  })

  it('displays Test Connection button', async () => {
    const user = userEvent.setup()
    render(<AddDeviceDialog />)

    await user.click(screen.getByRole('button', { name: /Add Device/i }))

    expect(screen.getByRole('button', { name: /Test Connection/i })).toBeInTheDocument()
  })

  it('shows helpful description text', async () => {
    const user = userEvent.setup()
    render(<AddDeviceDialog />)

    await user.click(screen.getByRole('button', { name: /Add Device/i }))

    expect(
      screen.getByText(/Add a new device to your homelab/i)
    ).toBeInTheDocument()
    expect(
      screen.getByText(/Test the connection before adding the device/i)
    ).toBeInTheDocument()
  })

  it('includes Cancel and submit buttons in footer', async () => {
    const user = userEvent.setup()
    render(<AddDeviceDialog />)

    await user.click(screen.getByRole('button', { name: /^Add Device$/i }))

    // Wait for dialog to open and verify footer buttons
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Cancel/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /^Add Device$/i })).toBeInTheDocument()
    })
  })

  it('allows filling out the form', async () => {
    const user = userEvent.setup()
    render(<AddDeviceDialog />)

    await user.click(screen.getByRole('button', { name: /^Add Device$/i }))

    // Fill out form fields
    await user.type(screen.getByLabelText(/Device Name/i), 'Test Server')
    await user.type(screen.getByLabelText(/IP Address/i), '192.168.1.100')
    await user.type(screen.getByLabelText(/MAC Address/i), '00:11:22:33:44:55')
    await user.type(screen.getByLabelText(/Username/i), 'admin')
    await user.type(screen.getByLabelText(/^Password$/i), 'password123')

    // Verify values
    expect(screen.getByLabelText(/Device Name/i)).toHaveValue('Test Server')
    expect(screen.getByLabelText(/IP Address/i)).toHaveValue('192.168.1.100')
    expect(screen.getByLabelText(/Username/i)).toHaveValue('admin')
  })
})
