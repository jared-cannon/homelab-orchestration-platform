// Shared utility functions for dependency handling and formatting

export function formatTime(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  return `${minutes} min`
}

export function formatRAM(mb: number): string {
  return mb >= 1024 ? `${(mb / 1024).toFixed(1)}GB` : `${mb}MB`
}

export function getDependencyIcon(type: string): string {
  switch (type) {
    case 'reverse_proxy':
      return '🔀'
    case 'database':
      return '🗄️'
    case 'cache':
      return '⚡'
    case 'application':
      return '📦'
    default:
      return '🔧'
  }
}

export function getDependencyName(dep: any): string {
  if (dep.name) return dep.name
  if (dep.engine) return dep.engine
  return dep.type.replace('_', ' ')
}
