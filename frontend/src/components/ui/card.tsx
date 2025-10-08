import { cn } from '../../lib/utils'

interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode
  className?: string
}

export function Card({ children, className, ...props }: CardProps) {
  return (
    <div
      className={cn(
        'bg-card text-card-foreground rounded-lg border border-border shadow-sm hover:shadow-md transition-shadow',
        className
      )}
      {...props}
    >
      {children}
    </div>
  )
}

interface CardHeaderProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode
  className?: string
}

export function CardHeader({ children, className, ...props }: CardHeaderProps) {
  return (
    <div className={cn('p-6 pb-3', className)} {...props}>
      {children}
    </div>
  )
}

interface CardTitleProps extends React.HTMLAttributes<HTMLHeadingElement> {
  children: React.ReactNode
  className?: string
}

export function CardTitle({ children, className, ...props }: CardTitleProps) {
  return (
    <h3
      className={cn('text-lg font-semibold leading-none tracking-tight', className)}
      {...props}
    >
      {children}
    </h3>
  )
}

interface CardDescriptionProps extends React.HTMLAttributes<HTMLParagraphElement> {
  children: React.ReactNode
  className?: string
}

export function CardDescription({
  children,
  className,
  ...props
}: CardDescriptionProps) {
  return (
    <p
      className={cn('text-sm text-muted-foreground mt-1.5', className)}
      {...props}
    >
      {children}
    </p>
  )
}

interface CardContentProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode
  className?: string
}

export function CardContent({ children, className, ...props }: CardContentProps) {
  return (
    <div className={cn('p-6 pt-0', className)} {...props}>
      {children}
    </div>
  )
}

interface CardFooterProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode
  className?: string
}

export function CardFooter({ children, className, ...props }: CardFooterProps) {
  return (
    <div className={cn('p-6 pt-0 flex items-center gap-2', className)} {...props}>
      {children}
    </div>
  )
}
