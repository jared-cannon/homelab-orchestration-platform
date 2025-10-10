import { LucideIcon } from 'lucide-react'
import { AccordionContent, AccordionItem, AccordionTrigger } from '../ui/accordion'

interface PhaseAccordionProps {
  value: string
  icon: LucideIcon
  title: string
  description: string
  children: React.ReactNode
}

export function PhaseAccordion({
  value,
  icon: Icon,
  title,
  description,
  children,
}: PhaseAccordionProps) {
  return (
    <AccordionItem value={value} className="border rounded-lg px-4">
      <AccordionTrigger className="hover:no-underline">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
            <Icon className="w-4 h-4 text-primary" />
          </div>
          <div className="text-left">
            <div className="font-semibold">{title}</div>
            <div className="text-sm text-muted-foreground font-normal">{description}</div>
          </div>
        </div>
      </AccordionTrigger>
      <AccordionContent className="space-y-4 pt-4">{children}</AccordionContent>
    </AccordionItem>
  )
}
