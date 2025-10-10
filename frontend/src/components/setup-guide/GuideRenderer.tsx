import {
  HardDrive,
  Download,
  Usb,
  Terminal,
  Network,
  Shield,
  CheckCircle2,
  ExternalLink,
} from 'lucide-react'
import { Button } from '../ui/button'
import { Accordion } from '../ui/accordion'
import { PhaseAccordion } from './PhaseAccordion'
import { GuideSection } from './GuideSection'
import { InfoBox } from './InfoBox'
import { CodeBlock } from './CodeBlock'
import { CommandBox } from './CommandBox'
import type {
  GuideConfig,
  ContentItem,
  IconName,
  GuidePhase,
  GuideSection as GuideSectionType,
} from './types'

const iconMap = {
  HardDrive,
  Download,
  Usb, // Keep for future use
  Terminal,
  Network,
  Shield,
}

function renderContent(items: ContentItem[] | string): React.ReactNode {
  // Handle string content (for simple text in InfoBox)
  if (typeof items === 'string') {
    return <p className="text-sm">{items}</p>
  }

  return items.map((item, index) => {
    switch (item.type) {
      case 'text':
        return (
          <p key={index} className={item.className || 'text-sm text-muted-foreground'}>
            {item.text}
          </p>
        )

      case 'code':
        return (
          <CodeBlock
            key={index}
            code={item.code}
            language={item.language}
            copyLabel={item.copyLabel}
            className={item.className}
          />
        )

      case 'command':
        return (
          <CommandBox
            key={index}
            command={item.command}
            description={item.description}
            label={item.label}
          />
        )

      case 'infoBox':
        return (
          <InfoBox key={index} variant={item.variant} title={item.title}>
            {renderContent(item.content)}
          </InfoBox>
        )

      case 'list':
        const ListTag = item.ordered ? 'ol' : 'ul'
        return (
          <ListTag
            key={index}
            className={
              item.className ||
              `space-y-1 ${item.ordered ? 'ml-4 list-decimal' : ''} text-sm text-muted-foreground`
            }
          >
            {item.items.map((listItem, i) => (
              <li key={i}>{listItem}</li>
            ))}
          </ListTag>
        )

      case 'grid':
        // Map columns to actual Tailwind classes (JIT compiler needs full class names)
        const gridColsClass = {
          1: 'grid-cols-1',
          2: 'grid-cols-2',
          3: 'grid-cols-3',
          4: 'grid-cols-4',
        }[item.columns] || 'grid-cols-2'

        return (
          <div key={index} className={`grid ${gridColsClass} gap-4 text-sm`}>
            {item.items.map((gridItem, i) => (
              <div key={i}>
                {gridItem.title && <p className="font-medium mb-2">{gridItem.title}</p>}
                <ul className="space-y-1 text-muted-foreground">
                  {gridItem.content.map((line, j) => (
                    <li key={j}>{line}</li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        )

      case 'linkButton':
        return (
          <div key={index} className="flex items-center justify-between">
            <div>
              <p className="font-medium">{item.label}</p>
              {item.description && (
                <p className="text-sm text-muted-foreground">{item.description}</p>
              )}
            </div>
            <Button variant="outline" size="sm" asChild>
              <a href={item.href} target="_blank" rel="noopener noreferrer">
                Download
                <ExternalLink className="ml-2 w-4 h-4" />
              </a>
            </Button>
          </div>
        )

      case 'stepList':
        return (
          <ol key={index} className="space-y-3 text-sm">
            {item.items.map((step, i) => (
              <li key={i} className="flex gap-3">
                <span className="font-semibold text-primary">{step.label}</span>
                <div className="flex-1">
                  {typeof step.content === 'string' ? (
                    <span>{step.content}</span>
                  ) : (
                    renderContent(step.content)
                  )}
                </div>
              </li>
            ))}
          </ol>
        )

      case 'custom':
        // Handle special custom components
        if (item.component === 'loginTerminal') {
          return (
            <div key={index} className="bg-background border rounded p-3 font-mono text-sm space-y-2">
              <div className="text-muted-foreground">[your-server-name] login: _</div>
              <p className="text-xs text-muted-foreground">Type your username and press Enter (use the username you created during installation)</p>
              <div className="text-muted-foreground">Password: _</div>
              <p className="text-xs text-muted-foreground">
                Type your password (won\'t show as you type - this is normal for security)
              </p>
            </div>
          )
        }
        if (item.component === 'sshPrompt') {
          return (
            <div key={index} className="bg-background border rounded p-2 font-mono text-xs">
              The authenticity of host... can\'t be established.
              <br />
              Are you sure you want to continue? (yes/no)
            </div>
          )
        }
        if (item.component === 'rufusConfig') {
          return (
            <div key={index} className="bg-background border rounded p-3 space-y-1 font-mono text-xs">
              <div>Device: [Your USB drive]</div>
              <div>Boot selection: [Click SELECT, choose Ubuntu ISO]</div>
              <div>Partition scheme: GPT</div>
              <div>Target system: UEFI</div>
              <div>File system: FAT32</div>
            </div>
          )
        }
        if (item.component === 'wizardTable') {
          // Validate props structure
          const props = item.props as { rows?: Array<{ label: string; value: string | { text: string; note?: string } }> }
          if (!props?.rows || !Array.isArray(props.rows)) {
            console.error('wizardTable requires props.rows array')
            return null
          }
          return (
            <div key={index} className="space-y-3 text-sm">
              {props.rows.map((row, i) => (
                <div key={i} className="flex gap-3 pb-2 border-b last:border-0">
                  <span className="font-semibold min-w-[120px]">{row.label}</span>
                  {typeof row.value === 'string' ? (
                    <span className="text-muted-foreground">{row.value}</span>
                  ) : (
                    <div className="flex-1">
                      <p className="text-muted-foreground mb-1">{row.value.text}</p>
                      {row.value.note && (
                        <p className="text-xs text-muted-foreground">{row.value.note}</p>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )
        }
        return null

      default:
        return null
    }
  })
}

function renderSection(section: GuideSectionType, index: number) {
  return (
    <GuideSection key={index} title={section.title}>
      {renderContent(section.content)}
    </GuideSection>
  )
}

function renderPhase(phase: GuidePhase) {
  return (
    <PhaseAccordion
      key={phase.id}
      value={phase.id}
      icon={iconMap[phase.icon as IconName]}
      title={phase.title}
      description={phase.description}
    >
      {phase.sections.map(renderSection)}
    </PhaseAccordion>
  )
}

interface GuideRendererProps {
  config: GuideConfig
}

export function GuideRenderer({ config }: GuideRendererProps) {
  return (
    <div className="space-y-4">
      {config.warningMessage && (
        <InfoBox variant={config.warningMessage.variant} title={config.warningMessage.title}>
          {renderContent(config.warningMessage.content)}
        </InfoBox>
      )}

      {/* Guide Overview */}
      <div className="bg-muted/50 rounded-lg p-4">
        <div className="flex items-center justify-between mb-2">
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            Guide Overview
          </p>
          <p className="text-xs text-muted-foreground">
            {config.phases.length} phases total
          </p>
        </div>
        <div className="flex gap-1">
          {config.phases.map((phase, index) => (
            <div
              key={phase.id}
              className="flex-1 h-1.5 rounded-full bg-primary/20"
              title={phase.title}
              aria-label={`Phase ${index + 1}: ${phase.title}`}
            />
          ))}
        </div>
        <p className="text-xs text-muted-foreground mt-2">
          Expand each phase below and follow the steps in order
        </p>
      </div>

      <Accordion type="multiple" className="space-y-2">
        {config.phases.map(renderPhase)}
      </Accordion>

      {config.conclusion && (
        <div className="bg-gradient-to-br from-primary/5 to-primary/10 border border-primary/20 rounded-lg p-6 space-y-3">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="w-5 h-5 text-primary" />
            <h3 className="font-semibold text-lg">{config.conclusion.title}</h3>
          </div>
          <p className="text-sm text-muted-foreground">{config.conclusion.description}</p>
          <div className="bg-background/50 rounded p-3 space-y-1 text-sm">
            <p className="font-medium">What you'll need:</p>
            <ul className="text-muted-foreground space-y-0.5 ml-4">
              {config.conclusion.checklist.map((item, i) => (
                <li key={i}>• {item}</li>
              ))}
            </ul>
          </div>
        </div>
      )}
    </div>
  )
}
