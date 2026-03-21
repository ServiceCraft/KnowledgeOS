import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { RefreshCw, CheckCircle, XCircle, Clock } from 'lucide-react';
import { useSyncStatus } from '@/hooks/useSync';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';

export function SyncPage() {
  const { data, isLoading, isError } = useSyncStatus();

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message="Не удалось загрузить статус синхронизации." />;
  if (!data) return <ErrorState message="Нет данных о синхронизации." />;

  return (
    <div className="space-y-4 max-w-2xl">
      <h1 className="text-2xl font-semibold">Статус синхронизации</h1>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Подписка
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              {data.subscription_active ? (
                <>
                  <CheckCircle className="h-5 w-5 text-green-500" />
                  <Badge variant="secondary" className="bg-green-100 text-green-800">
                    Активна
                  </Badge>
                </>
              ) : (
                <>
                  <XCircle className="h-5 w-5 text-destructive" />
                  <Badge variant="destructive">Неактивна</Badge>
                </>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Последняя синхронизация
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-muted-foreground" />
              <span className="text-sm">
                {data.last_sync_at
                  ? new Date(data.last_sync_at).toLocaleString()
                  : 'Ещё не синхронизировано'}
              </span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Последний результат
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <RefreshCw className="h-5 w-5 text-muted-foreground" />
              <span className="text-sm">
                {data.last_sync_result ?? 'Н/Д'}
              </span>
            </div>
          </CardContent>
        </Card>

        {data.last_error && (
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-destructive">
                Последняя ошибка
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-destructive">{data.last_error}</p>
            </CardContent>
          </Card>
        )}
      </div>

      <p className="text-xs text-muted-foreground">
        Статус обновляется автоматически каждые 60 секунд.
      </p>
    </div>
  );
}
