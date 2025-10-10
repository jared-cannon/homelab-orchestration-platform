export type InfoBoxVariant = 'warning' | 'info' | 'success' | 'tip'

export type IconName =
  | 'HardDrive'
  | 'Download'
  | 'Usb'
  | 'Terminal'
  | 'Network'
  | 'Shield'

export interface TextContent {
  type: 'text'
  text: string
  className?: string
}

export interface CodeBlockContent {
  type: 'code'
  code: string
  language?: 'bash' | 'powershell' | 'text'
  copyLabel?: string
  className?: string
}

export interface CommandContent {
  type: 'command'
  command: string
  description?: string
  label?: string
}

export interface InfoBoxContent {
  type: 'infoBox'
  variant: InfoBoxVariant
  title?: string
  content: string | ContentItem[]
}

export interface ListContent {
  type: 'list'
  ordered?: boolean
  items: string[]
  className?: string
}

export interface GridContent {
  type: 'grid'
  columns: number
  items: Array<{
    title?: string
    content: string[]
  }>
}

export interface LinkButtonContent {
  type: 'linkButton'
  href: string
  label: string
  description?: string
}

export interface StepListContent {
  type: 'stepList'
  items: Array<{
    label: string
    content: string | ContentItem[]
  }>
}

export interface CustomContent {
  type: 'custom'
  component: string // For special cases like the login terminal
  props?: Record<string, unknown>
}

export type ContentItem =
  | TextContent
  | CodeBlockContent
  | CommandContent
  | InfoBoxContent
  | ListContent
  | GridContent
  | LinkButtonContent
  | StepListContent
  | CustomContent

export interface GuideSection {
  title: string
  content: ContentItem[]
}

export interface GuidePhase {
  id: string
  icon: IconName
  title: string
  description: string
  sections: GuideSection[]
}

export interface GuideConfig {
  title: string
  description: string
  warningMessage?: InfoBoxContent
  phases: GuidePhase[]
  conclusion?: {
    title: string
    description: string
    checklist: string[]
  }
}
