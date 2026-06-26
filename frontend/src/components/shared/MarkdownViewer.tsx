import MDEditor from '@uiw/react-md-editor';
import { useUIStore } from '@/stores/uiStore';

interface MarkdownViewerProps {
  source: string;
}

export function MarkdownViewer({ source }: MarkdownViewerProps) {
  const darkMode = useUIStore((s) => s.darkMode);

  return (
    <div data-color-mode={darkMode ? 'dark' : 'light'}>
      <MDEditor.Markdown source={source} />
    </div>
  );
}
