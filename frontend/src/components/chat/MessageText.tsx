import { cn } from '@/lib/utils';

interface MessageTextProps {
  text: string;
  className?: string;
}

function plainTextForClipboard(text: string): string {
  return text.replace(/\n+$/, '').replace(/^\n+/, '');
}

export function MessageText({ text, className }: MessageTextProps) {
  function handleCopy(event: React.ClipboardEvent<HTMLDivElement>) {
    event.preventDefault();
    event.clipboardData.setData('text/plain', plainTextForClipboard(text));
  }

  return (
    <div
      className={cn('select-text whitespace-pre-wrap break-words', className)}
      onCopy={handleCopy}
    >
      {text}
    </div>
  );
}
