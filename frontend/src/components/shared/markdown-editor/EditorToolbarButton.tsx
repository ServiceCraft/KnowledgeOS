import type { ReactNode } from 'react';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';

interface EditorToolbarButtonProps {
  icon: ReactNode;
  label: string;
  shortcut?: string;
  disabled?: boolean;
  onClick: () => void;
  className?: string;
}

export function EditorToolbarButton({
  icon,
  label,
  shortcut,
  disabled,
  onClick,
  className,
}: EditorToolbarButtonProps) {
  const tooltip = shortcut ? `${label} (${shortcut})` : label;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            disabled={disabled}
            onClick={(e) => {
              e.stopPropagation();
              onClick();
            }}
            aria-label={label}
            className={cn('shrink-0', className)}
          >
            {icon}
          </Button>
        }
      />
      <TooltipContent side="bottom">{tooltip}</TooltipContent>
    </Tooltip>
  );
}
