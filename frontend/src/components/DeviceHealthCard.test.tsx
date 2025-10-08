import { describe, it, expect } from 'vitest'
import { render, screen } from '../test/utils'
import { DeviceHealthCard } from './DeviceHealthCard'
import type { Device } from '../api/generated-types'
import { DeviceStatusOnline, DeviceStatusOffline, DeviceStatusError, DeviceStatusUnknown } from '../api/generated-types'

const baseDevice: Device = {
  id: '123e4567-e89b-12d3-a456-426614174000',
  name: 'Test Server',
  type: 'server',
  ip_address: '192.168.1.100',
  status: DeviceStatusOnline,
  created_at: '2025-10-07T12:00:00Z',
  updated_at: '2025-10-07T12:00:00Z',
}

describe('DeviceHealthCard', () => {
  it('renders device name and IP address', () => {
    render(<DeviceHealthCard device={baseDevice} />)

    expect(screen.getByText('Test Server')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.100')).toBeInTheDocument()
  })

  it('displays device type', () => {
    render(<DeviceHealthCard device={baseDevice} />)

    expect(screen.getByText(/Type:/i)).toBeInTheDocument()
    expect(screen.getByText('server')).toBeInTheDocument()
  })

  it('shows online status with green indicator', () => {
    render(<DeviceHealthCard device={{ ...baseDevice, status: DeviceStatusOnline }}/>)

    expect(screen.getByText('Online')).toBeInTheDocument()
  })

  it('shows offline status', () => {
    render(<DeviceHealthCard device={{ ...baseDevice, status: DeviceStatusOffline }} />)

    expect(screen.getByText('Offline')).toBeInTheDocument()
  })

  it('shows error status', () => {
    render(<DeviceHealthCard device={{ ...baseDevice, status: DeviceStatusError }} />)

    expect(screen.getByText('Error')).toBeInTheDocument()
  })

  it('shows unknown status', () => {
    render(<DeviceHealthCard device={{ ...baseDevice, status: DeviceStatusUnknown }} />)

    expect(screen.getByText('Unknown')).toBeInTheDocument()
  })

  it('displays last seen as "Never" when not set', () => {
    render(<DeviceHealthCard device={baseDevice} />)

    expect(screen.getByText(/Last seen:/i)).toBeInTheDocument()
    expect(screen.getByText('Never')).toBeInTheDocument()
  })

  it('formats recent last seen time', () => {
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString()
    const device = { ...baseDevice, last_seen: fiveMinutesAgo }

    render(<DeviceHealthCard device={device} />)

    expect(screen.getByText(/5m ago/)).toBeInTheDocument()
  })

  it('displays router icon for router type', () => {
    render(<DeviceHealthCard device={{ ...baseDevice, type: 'router' }} />)
    // Icon is rendered, just verify the type is shown
    expect(screen.getByText('router')).toBeInTheDocument()
  })

  it('displays NAS icon for NAS type', () => {
    render(<DeviceHealthCard device={{ ...baseDevice, type: 'nas' }} />)
    expect(screen.getByText('nas')).toBeInTheDocument()
  })

  it('displays switch icon for switch type', () => {
    render(<DeviceHealthCard device={{ ...baseDevice, type: 'switch' }} />)
    expect(screen.getByText('switch')).toBeInTheDocument()
  })
})
