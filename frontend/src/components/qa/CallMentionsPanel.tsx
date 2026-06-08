import { useState } from 'react';
import { Phone } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useQACallMentions } from '@/hooks/useCalls';
import { CallTranscriptSheet } from '@/components/calls/CallTranscriptSheet';
import type { CallMention } from '@/types';

interface CallMentionsPanelProps {
  qaId: string;
}

export function CallMentionsPanel({ qaId }: CallMentionsPanelProps) {
  const { data, isLoading, isError } = useQACallMentions(qaId);
  const [selected, setSelected] = useState<CallMention | null>(null);

  const mentions = data?.data ?? [];
  const total = data?.total ?? 0;

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Phone className="h-4 w-4" />
            Упоминания в звонках
            {total > 0 && (
              <span className="text-muted-foreground font-normal">({total})</span>
            )}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : isError ? (
            <p className="text-sm text-destructive">
              Не удалось загрузить список упоминаний.
            </p>
          ) : mentions.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Этот вопрос пока не упоминался ни в одном звонке.
            </p>
          ) : (
            <ul className="divide-y">
              {mentions.map((m) => (
                <li key={m.id}>
                  <button
                    type="button"
                    onClick={() => setSelected(m)}
                    className="w-full text-left py-3 px-2 -mx-2 rounded hover:bg-muted/50 transition-colors"
                  >
                    <div className="flex items-center justify-between text-xs text-muted-foreground mb-1 gap-2">
                      <span className="truncate">
                        {m.call_title || 'Звонок без названия'}
                      </span>
                      {m.call_occurred_at && (
                        <time className="shrink-0">
                          {new Date(m.call_occurred_at).toLocaleDateString('ru-RU')}
                        </time>
                      )}
                    </div>
                    <p className="text-sm line-clamp-2">{m.snippet}</p>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>

      <CallTranscriptSheet
        open={!!selected}
        onOpenChange={(open) => !open && setSelected(null)}
        callId={selected?.call_id ?? null}
        highlightStart={selected?.start_offset ?? undefined}
        highlightEnd={selected?.end_offset ?? undefined}
      />
    </>
  );
}
