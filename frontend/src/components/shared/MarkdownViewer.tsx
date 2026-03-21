import MDEditor from '@uiw/react-md-editor';

interface MarkdownViewerProps {
  source: string;
}

export function MarkdownViewer({ source }: MarkdownViewerProps) {
  return (
    <div data-color-mode="light">
      <MDEditor.Markdown source={source} />
    </div>
  );
}
