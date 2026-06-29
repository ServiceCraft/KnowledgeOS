import type { ReactNode } from 'react';
import { Label } from '@/components/ui/label';
import { HelpCircle } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';

export function FieldHint({
  label,
  htmlFor,
  hint,
  className,
}: {
  label: ReactNode;
  htmlFor?: string;
  hint: string;
  className?: string;
}) {
  return (
    <div className={cn('flex items-center gap-1.5', className)}>
      {htmlFor ? <Label htmlFor={htmlFor}>{label}</Label> : <span className="text-sm font-medium">{label}</span>}
      <Tooltip>
        <TooltipTrigger
          render={
            <button
              type="button"
              className="inline-flex shrink-0 text-muted-foreground transition hover:text-foreground"
              aria-label="Подсказка"
            >
              <HelpCircle className="h-3.5 w-3.5" />
            </button>
          }
        />
        <TooltipContent side="top" className="max-w-sm p-3 text-xs leading-relaxed whitespace-normal">
          {hint}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}
