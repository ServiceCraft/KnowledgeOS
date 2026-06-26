import MDEditor from '@uiw/react-md-editor';
import '@uiw/react-markdown-preview/markdown.css';
import { useUIStore } from '@/stores/uiStore';
import { cn } from '@/lib/utils';

interface MarkdownViewerProps {
  source: string;
  className?: string;
}

export function MarkdownViewer({ source, className }: MarkdownViewerProps) {
  const darkMode = useUIStore((s) => s.darkMode);

  return (
    <div
      data-color-mode={darkMode ? 'dark' : 'light'}
      className={cn('article-markdown', className)}
    >
      <MDEditor.Markdown source={source} style={{ backgroundColor: 'transparent' }} />
    </div>
  );
}
