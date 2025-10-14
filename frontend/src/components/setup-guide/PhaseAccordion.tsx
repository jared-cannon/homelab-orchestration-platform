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
    <AccordionItem value={value} className="border rounded-lg px-5 bg-card/50">
      <AccordionTrigger className="hover:no-underline py-5">
        <div className="flex items-center gap-4">
          <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
            <Icon className="w-5 h-5 text-primary" />
          </div>
          <div className="text-left">
            <div className="font-semibold text-base mb-1">{title}</div>
            <div className="text-sm text-muted-foreground font-normal">{description}</div>
          </div>
        </div>
      </AccordionTrigger>
      <AccordionContent className="space-y-6 pt-2 pb-6">{children}</AccordionContent>
    </AccordionItem>
  )
}
