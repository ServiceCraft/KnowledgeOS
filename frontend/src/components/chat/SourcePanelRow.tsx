import { Link } from 'react-router-dom';
import { ExternalLink } from 'lucide-react';
import type { ChatSource } from '@/types';
import { Badge } from '@/components/ui/badge';
import { buildSourceUrl, sourcePreviewText, SOURCE_TYPE_LABELS } from '@/lib/chatSources';
import { cn } from '@/lib/utils';

interface SourcePanelRowProps {
  source: ChatSource;
  className?: string;
}

export function SourcePanelRow({ source, className }: SourcePanelRowProps) {
  const url = buildSourceUrl(source.entity_type, source.entity_id);
  const typeLabel = SOURCE_TYPE_LABELS[source.entity_type] ?? source.entity_type;
  const preview = sourcePreviewText(source);
  const score = Number.isFinite(source.score) ? Math.round(source.score * 100) : null;

  const content = (
    <div className={cn('rounded-lg border bg-card p-3 text-sm transition hover:bg-muted/60', className)}>
      <div className="flex items-center gap-2">
        <Badge variant="outline" className="shrink-0 text-[10px] uppercase">
          {typeLabel}
        </Badge>
        <span className="min-w-0 flex-1 truncate font-medium">{source.title || source.source_id}</span>
        {score !== null && <span className="text-xs text-muted-foreground">{score}%</span>}
        {url && <ExternalLink className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />}
      </div>
      {preview && <p className="mt-2 line-clamp-3 text-xs text-muted-foreground">{preview}</p>}
    </div>
  );

  if (!url) {
    return content;
  }

  return (
    <Link to={url} className="block">
      {content}
    </Link>
  );
}
