import { Heading, ChevronDown } from 'lucide-react';
import type { ICommand } from '@uiw/react-md-editor/commands';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';

interface HeadingOption {
  command: ICommand;
  label: string;
}

interface EditorHeadingMenuProps {
  disabled?: boolean;
  options: HeadingOption[];
  onSelect: (command: ICommand) => void;
}

export function EditorHeadingMenu({ disabled, options, onSelect }: EditorHeadingMenuProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        disabled={disabled}
        render={
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={disabled}
            className={cn('h-7 gap-1 px-2 text-xs font-medium')}
            onClick={(e) => e.stopPropagation()}
          />
        }
      >
        <Heading className="size-3.5" />
        Заголовок
        <ChevronDown className="size-3 opacity-60" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-40">
        {options.map((option) => (
          <DropdownMenuItem
            key={option.label}
            onClick={() => onSelect(option.command)}
          >
            {option.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
