import { cn } from '@/lib/utils';

export type PreviewMode = 'edit' | 'live' | 'preview';

const MODES: { value: PreviewMode; label: string }[] = [
  { value: 'edit', label: 'Редактор' },
  { value: 'live', label: 'Разделённый' },
  { value: 'preview', label: 'Просмотр' },
];

interface EditorPreviewTabsProps {
  value: PreviewMode;
  onChange: (mode: PreviewMode) => void;
}

export function EditorPreviewTabs({ value, onChange }: EditorPreviewTabsProps) {
  return (
    <div
      className="inline-flex h-8 items-center rounded-lg bg-muted p-[3px]"
      onClick={(e) => e.stopPropagation()}
    >
      {MODES.map((mode) => (
        <button
          key={mode.value}
          type="button"
          aria-pressed={value === mode.value}
          className={cn(
            'inline-flex h-[calc(100%-1px)] items-center rounded-md px-2.5 text-xs font-medium whitespace-nowrap transition-all',
            'text-foreground/60 hover:text-foreground',
            value === mode.value && 'bg-background text-foreground shadow-sm'
          )}
          onClick={() => onChange(mode.value)}
        >
          {mode.label}
        </button>
      ))}
    </div>
  );
}
