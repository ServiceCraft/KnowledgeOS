import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { useRagIndexStatus, useRagReindex, useRagSearch } from '@/hooks/useRag';
import { toast } from 'sonner';
import { Loader2, RefreshCw } from 'lucide-react';

export function RagTab() {
  const { data: status, isLoading, isError } = useRagIndexStatus();
  const reindex = useRagReindex();
  const search = useRagSearch();
  const [query, setQuery] = useState('');

  const runSearch = () => {
    if (!query.trim()) return;
    search.mutate({ query: query.trim(), hybrid_top_k: 10 });
  };

  return (
    <div className="space-y-6 max-w-3xl">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Индексация</CardTitle>
          <CardDescription>Статус векторного индекса базы знаний</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {isError ? (
            <ErrorState message="Не удалось загрузить статус индекса." />
          ) : isLoading ? (
            <LoadingState />
          ) : status ? (
            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <p className="text-muted-foreground">Чанков в индексе</p>
                <p className="text-2xl font-semibold">{status.embeddings}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Задачи в очереди</p>
                <div className="mt-1 space-y-1">
                  {Object.entries(status.jobs ?? {}).map(([k, v]) => (
                    <p key={k}>{k}: {v}</p>
                  ))}
                </div>
              </div>
            </div>
          ) : null}
          <Button
            variant="outline"
            onClick={() => reindex.mutate(undefined, {
              onSuccess: () => toast.success('Переиндексация поставлена в очередь'),
              onError: () => toast.error('Не удалось запустить переиндексацию'),
            })}
            disabled={reindex.isPending}
          >
            {reindex.isPending ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : <RefreshCw className="h-4 w-4 mr-2" />}
            Переиндексировать
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Отладочный поиск</CardTitle>
          <CardDescription>RAG-поиск по индексу (для администраторов)</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <Input
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Запрос..."
              onKeyDown={(e) => e.key === 'Enter' && runSearch()}
            />
            <Button onClick={runSearch} disabled={search.isPending || !query.trim()}>
              {search.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Искать'}
            </Button>
          </div>
          {search.data?.results && search.data.results.length > 0 && (
            <div className="space-y-2">
              {search.data.results.map((r) => (
                <div key={r.source_id} className="border rounded-md p-3 text-sm">
                  <div className="flex items-center gap-2 mb-1">
                    <Badge variant="outline">{r.entity_type}</Badge>
                    <span className="font-medium">{r.title}</span>
                    <span className="text-muted-foreground ml-auto">оценка {r.score.toFixed(3)}</span>
                  </div>
                  <p className="text-muted-foreground line-clamp-2">{r.snippet || r.content}</p>
                </div>
              ))}
            </div>
          )}
          {search.data?.results?.length === 0 && (
            <p className="text-sm text-muted-foreground">Ничего не найдено.</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
