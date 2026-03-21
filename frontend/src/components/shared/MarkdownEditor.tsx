import MDEditor from '@uiw/react-md-editor';

interface MarkdownEditorProps {
  value: string;
  onChange: (value: string) => void;
  height?: number;
}

export function MarkdownEditor({ value, onChange, height = 400 }: MarkdownEditorProps) {
  return (
    <div data-color-mode="light" className="[.dark_&]:hidden">
      <MDEditor value={value} onChange={(v) => onChange(v ?? '')} height={height} />
    </div>
  );
}
