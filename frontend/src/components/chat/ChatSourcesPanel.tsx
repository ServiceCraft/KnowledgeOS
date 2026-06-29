import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { SourcePanelRow } from '@/components/chat/SourcePanelRow';
import type { ChatSource } from '@/types';

interface ChatSourcesPanelProps {
  sources: ChatSource[];
  title?: string;
  description?: string;
  emptyMessage?: string;
  className?: string;
}

export function ChatSourcesPanel({
  sources,
  title = 'Источники',
  description = 'Материалы, использованные в ответах текущего диалога',
  emptyMessage = 'Источники появятся после ответа с RAG-контекстом.',
  className,
}: ChatSourcesPanelProps) {
  return (
    <Card className={className ?? 'hidden w-80 shrink-0 xl:flex xl:flex-col'}>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent className="min-h-0 flex-1 overflow-y-auto space-y-2">
        {sources.map((source) => (
          <SourcePanelRow key={source.source_id} source={source} />
        ))}
        {sources.length === 0 && (
          <p className="text-sm text-muted-foreground">{emptyMessage}</p>
        )}
      </CardContent>
    </Card>
  );
}
