import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'

const badgeVariants = cva(
  'inline-flex items-center rounded-md px-2 py-1 text-xs font-medium ring-1 ring-inset',
  {
    variants: {
      variant: {
        default:
          'bg-indigo-500/10 text-indigo-700 dark:text-indigo-400 ring-indigo-500/20',
        secondary:
          'bg-slate-500/10 text-slate-700 dark:text-slate-400 ring-slate-500/20',
        success:
          'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 ring-emerald-500/20',
        warning:
          'bg-amber-500/10 text-amber-700 dark:text-amber-400 ring-amber-500/20',
        danger:
          'bg-red-500/10 text-red-700 dark:text-red-400 ring-red-500/20',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return <div className={badgeVariants({ variant, className })} {...props} />
}

export { Badge, badgeVariants }
