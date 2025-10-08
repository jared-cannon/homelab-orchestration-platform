import { describe, it, expect } from 'vitest'
import { render, screen } from '../test/utils'
import { FirstRunWizard } from './FirstRunWizard'

describe('FirstRunWizard', () => {
  it('renders welcome message', () => {
    render(<FirstRunWizard />)

    expect(screen.getByText(/Welcome to Your Homelab!/i)).toBeInTheDocument()
  })

  it('shows clear instructions about what is needed', () => {
    render(<FirstRunWizard />)

    expect(screen.getByText(/Let's get started by adding your first device/i)).toBeInTheDocument()
  })

  it('displays checklist of requirements', () => {
    render(<FirstRunWizard />)

    // Check for the "What you'll need" section
    expect(screen.getByText(/What you'll need:/i)).toBeInTheDocument()

    // Verify each requirement is listed
    expect(screen.getByText(/The device's IP address/i)).toBeInTheDocument()
    expect(screen.getByText(/Login credentials \(username and password\)/i)).toBeInTheDocument()
    expect(screen.getByText(/The device should be powered on/i)).toBeInTheDocument()
  })

  it('includes Add Device button', () => {
    render(<FirstRunWizard />)

    // The AddDeviceDialog renders a button with "Add Device" text
    expect(screen.getByRole('button', { name: /Add Device/i })).toBeInTheDocument()
  })

  it('shows help link for finding IP address', () => {
    render(<FirstRunWizard />)

    expect(screen.getByText(/Not sure how to find your device's IP address\?/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Get help/i })).toBeInTheDocument()
  })

  it('mentions next steps after device is added', () => {
    render(<FirstRunWizard />)

    expect(screen.getByText(/Once your device is added/i)).toBeInTheDocument()
    expect(screen.getByText(/deploy apps like Nextcloud/i)).toBeInTheDocument()
  })
})
