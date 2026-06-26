import { Link } from 'react-router-dom';
import { ExternalLink } from 'lucide-react';
import type { ChatSource } from '@/types';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  buildSourceUrl,
  formatMessageBody,
  resolveCitedSources,
  sourcePreviewText,
  SOURCE_TYPE_LABELS,
  type ChatAnswerInput,
} from '@/lib/chatSources';
import { cn } from '@/lib/utils';
import { MessageText } from '@/components/chat/MessageText';

interface ChatCitationsProps {
  sources: ChatSource[];
  citedSourceIds?: string[];
  content?: string;
  className?: string;
}

export function ChatCitations({ sources, citedSourceIds, content = '', className }: ChatCitationsProps) {
  const cited = resolveCitedSources(content, sources, citedSourceIds);
  if (cited.length === 0) return null;

  return (
    <div className={cn('mt-2 space-y-1.5 border-t border-border/60 pt-2 select-none', className)}>
      <p className="text-xs font-medium text-muted-foreground">Источники</p>
      <ul className="space-y-1">
        {cited.map((source) => (
          <CitationLink key={source.source_id} source={source} />
        ))}
      </ul>
    </div>
  );
}

function CitationLink({ source }: { source: ChatSource }) {
  const url = buildSourceUrl(source.entity_type, source.entity_id);
  const label = source.title || source.source_id;
  const preview = sourcePreviewText(source);
  const typeLabel = SOURCE_TYPE_LABELS[source.entity_type] ?? source.entity_type;

  const inner = (
    <>
      <Badge variant="outline" className="shrink-0 text-[10px] uppercase">
        {typeLabel}
      </Badge>
      <span className="truncate underline-offset-2 group-hover:underline">{label}</span>
      {url && <ExternalLink className="h-3 w-3 shrink-0 opacity-60" />}
    </>
  );

  const linkClass =
    'group inline-flex max-w-full items-center gap-1.5 rounded px-1 py-0.5 text-xs text-primary hover:bg-primary/5';

  const trigger = url ? (
    <Link to={url} className={linkClass}>
      {inner}
    </Link>
  ) : (
    <span className={cn(linkClass, 'cursor-default text-foreground')}>{inner}</span>
  );

  if (!preview) {
    return <li>{trigger}</li>;
  }

  return (
    <li>
      <Tooltip>
        <TooltipTrigger render={trigger} />
        <TooltipContent side="top" className="max-w-sm p-3 text-xs leading-relaxed">
          <p className="max-h-48 overflow-y-auto whitespace-pre-wrap">{preview}</p>
        </TooltipContent>
      </Tooltip>
    </li>
  );
}

interface ChatAnswerContentProps {
  message: ChatAnswerInput;
  className?: string;
  showCitations?: boolean;
}

export function ChatAnswerContent({ message, className, showCitations = true }: ChatAnswerContentProps) {
  const body = formatMessageBody(message.content);

  return (
    <div className={className}>
      {body && <MessageText text={body} />}
      {showCitations && (
        <ChatCitations
          sources={message.sources}
          citedSourceIds={message.cited_source_ids}
          content={message.content}
        />
      )}
    </div>
  );
}
