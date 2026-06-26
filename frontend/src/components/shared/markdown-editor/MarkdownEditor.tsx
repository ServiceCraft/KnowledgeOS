import { useMemo, useState } from 'react';
import MDEditor from '@uiw/react-md-editor';
import '@uiw/react-md-editor/markdown-editor.css';
import '@uiw/react-markdown-preview/markdown.css';
import './markdown-editor.theme.css';
import { useUIStore } from '@/stores/uiStore';
import { TooltipProvider } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';
import { markdownPreviewOptions } from '@/lib/markdown';
import {
  createArticleEditorCommands,
  createPreviewTabsCommand,
} from './editorCommands';
import type { PreviewMode } from './EditorPreviewTabs';

const DEFAULT_HEIGHT = 480;

interface MarkdownEditorProps {
  value: string;
  onChange: (value: string) => void;
  height?: number;
  className?: string;
}

export function MarkdownEditor({
  value,
  onChange,
  height = DEFAULT_HEIGHT,
  className,
}: MarkdownEditorProps) {
  const darkMode = useUIStore((s) => s.darkMode);
  const [previewMode, setPreviewMode] = useState<PreviewMode>('live');

  const commands = useMemo(() => createArticleEditorCommands(), []);
  const extraCommands = useMemo(
    () => [createPreviewTabsCommand(previewMode, setPreviewMode)],
    [previewMode]
  );

  return (
    <TooltipProvider>
      <div
        data-color-mode={darkMode ? 'dark' : 'light'}
        className={cn('markdown-editor space-y-1.5', className)}
      >
        <MDEditor
          value={value}
          onChange={(v) => onChange(v ?? '')}
          height={height}
          preview={previewMode}
          commands={commands}
          extraCommands={extraCommands}
          className="markdown-editor-surface"
          textareaProps={{ placeholder: 'Начните писать...' }}
          previewOptions={{ className: 'article-markdown', ...markdownPreviewOptions }}
        />
        <p className="text-xs text-muted-foreground">Поддерживается Markdown</p>
      </div>
    </TooltipProvider>
  );
}
