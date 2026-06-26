import MDEditor from '@uiw/react-md-editor';
import { useUIStore } from '@/stores/uiStore';

interface MarkdownEditorProps {
  value: string;
  onChange: (value: string) => void;
  height?: number;
}

export function MarkdownEditor({ value, onChange, height = 400 }: MarkdownEditorProps) {
  const darkMode = useUIStore((s) => s.darkMode);

  return (
    <div data-color-mode={darkMode ? 'dark' : 'light'}>
      <MDEditor value={value} onChange={(v) => onChange(v ?? '')} height={height} />
    </div>
  );
}
