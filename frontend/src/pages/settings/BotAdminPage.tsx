import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import {
  useBotSettings,
  useUpdateBotSettings,
  useBotSecrets,
  useSetBotSecret,
  useDeleteBotSecret,
} from '@/hooks/useBotAdmin';
import { useRagIndexStatus, useRagReindex, useRagSearch } from '@/hooks/useRag';
import type { BotSettings, SecretKind, TenantSecretStatus } from '@/types';
import { toast } from 'sonner';
import { Loader2, RefreshCw } from 'lucide-react';

const SECRET_LABELS: Record<SecretKind, string> = {
  llm: 'LLM API',
  telegram: 'Telegram',
  max: 'MAX',
  vk: 'VK',
  bitrix24: 'Bitrix24',
};

function SettingsTab() {
  const { data, isLoading, isError } = useBotSettings();
  const update = useUpdateBotSettings();
  const [form, setForm] = useState<Partial<BotSettings>>({});

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  if (isLoading) return <LoadingState />;
  if (isError || !data) return <ErrorState message="Не удалось загрузить настройки бота." />;

  const save = () => {
    update.mutate(
      {
        enabled: form.enabled,
        provider: form.provider,
        model_tier: form.model_tier,
        model: form.model || undefined,
        temperature: form.temperature,
        max_tokens: form.max_tokens,
        persona_name: form.persona_name,
        persona_tone: form.persona_tone,
        persona_rules: form.persona_rules,
        min_retrieval_score: form.min_retrieval_score,
        min_confidence: form.min_confidence,
        escalate_on_low_confidence: form.escalate_on_low_confidence,
        require_citations: form.require_citations,
        allowed_theme_ids: Array.isArray(form.allowed_theme_ids) ? form.allowed_theme_ids : [],
      },
      {
        onSuccess: () => toast.success('Настройки сохранены'),
        onError: () => toast.error('Не удалось сохранить настройки'),
      }
    );
  };

  return (
    <div className="space-y-6 max-w-2xl">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Основные</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3">
            <Label htmlFor="enabled">Бот включён</Label>
            <Select
              value={form.enabled ? 'true' : 'false'}
              onValueChange={(v) => setForm((f) => ({ ...f, enabled: v === 'true' }))}
            >
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="true">Да</SelectItem>
                <SelectItem value="false">Нет</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Тариф модели</Label>
              <Select
                value={form.model_tier ?? 'lite'}
                onValueChange={(v) => setForm((f) => ({ ...f, model_tier: v as BotSettings['model_tier'] }))}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="lite">Lite</SelectItem>
                  <SelectItem value="pro">Pro</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="model">Модель</Label>
              <Input
                id="model"
                value={form.model ?? ''}
                onChange={(e) => setForm((f) => ({ ...f, model: e.target.value }))}
                placeholder="По умолчанию"
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="temperature">Temperature</Label>
              <Input
                id="temperature"
                type="number"
                step="0.1"
                min="0"
                max="2"
                value={form.temperature ?? 0.2}
                onChange={(e) => setForm((f) => ({ ...f, temperature: parseFloat(e.target.value) }))}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="max_tokens">Max tokens</Label>
              <Input
                id="max_tokens"
                type="number"
                value={form.max_tokens ?? 1024}
                onChange={(e) => setForm((f) => ({ ...f, max_tokens: parseInt(e.target.value, 10) }))}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Персона</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="persona_name">Имя</Label>
            <Input
              id="persona_name"
              value={form.persona_name ?? ''}
              onChange={(e) => setForm((f) => ({ ...f, persona_name: e.target.value }))}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="persona_tone">Тон</Label>
            <Input
              id="persona_tone"
              value={form.persona_tone ?? ''}
              onChange={(e) => setForm((f) => ({ ...f, persona_tone: e.target.value }))}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="persona_rules">Правила</Label>
            <Textarea
              id="persona_rules"
              value={form.persona_rules ?? ''}
              onChange={(e) => setForm((f) => ({ ...f, persona_rules: e.target.value }))}
              rows={4}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Guardrails</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="min_retrieval">Min retrieval score</Label>
              <Input
                id="min_retrieval"
                type="number"
                step="0.01"
                value={form.min_retrieval_score ?? 0}
                onChange={(e) => setForm((f) => ({ ...f, min_retrieval_score: parseFloat(e.target.value) }))}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="min_confidence">Min confidence</Label>
              <Input
                id="min_confidence"
                type="number"
                step="0.01"
                value={form.min_confidence ?? 0}
                onChange={(e) => setForm((f) => ({ ...f, min_confidence: parseFloat(e.target.value) }))}
              />
            </div>
          </div>
          <div className="flex flex-wrap gap-4">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.escalate_on_low_confidence ?? false}
                onChange={(e) => setForm((f) => ({ ...f, escalate_on_low_confidence: e.target.checked }))}
              />
              Эскалация при низкой уверенности
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.require_citations ?? false}
                onChange={(e) => setForm((f) => ({ ...f, require_citations: e.target.checked }))}
              />
              Требовать цитирование
            </label>
          </div>
        </CardContent>
      </Card>

      <Button onClick={save} disabled={update.isPending}>
        {update.isPending ? 'Сохранение...' : 'Сохранить настройки'}
      </Button>
    </div>
  );
}

function SecretsTab() {
  const { data, isLoading, isError } = useBotSecrets();
  const setSecret = useSetBotSecret();
  const deleteSecret = useDeleteBotSecret();
  const [editing, setEditing] = useState<TenantSecretStatus | null>(null);
  const [value, setValue] = useState('');
  const [deleteKind, setDeleteKind] = useState<SecretKind | null>(null);

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message="Не удалось загрузить секреты." />;

  const statuses = data ?? [];

  const handleSave = () => {
    if (!editing || !value.trim()) return;
    setSecret.mutate(
      { kind: editing.kind, data: { value } },
      {
        onSuccess: () => {
          setEditing(null);
          setValue('');
          toast.success('Секрет сохранён');
        },
        onError: () => toast.error('Не удалось сохранить секрет'),
      }
    );
  };

  const handleDelete = () => {
    if (!deleteKind) return;
    deleteSecret.mutate(deleteKind, {
      onSuccess: () => {
        setDeleteKind(null);
        toast.success('Секрет удалён');
      },
      onError: () => toast.error('Не удалось удалить секрет'),
    });
  };

  return (
    <div className="space-y-4 max-w-2xl">
      {statuses.map((s) => (
        <Card key={s.kind}>
          <CardContent className="pt-6 flex items-center justify-between gap-4">
            <div>
              <p className="font-medium">{SECRET_LABELS[s.kind] ?? s.kind}</p>
              <div className="flex items-center gap-2 mt-1">
                <Badge variant={s.is_set ? 'secondary' : 'outline'}>
                  {s.is_set ? 'Настроен' : 'Не задан'}
                </Badge>
                {s.updated_at && (
                  <span className="text-xs text-muted-foreground">
                    {new Date(s.updated_at).toLocaleString()}
                  </span>
                )}
              </div>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={() => { setEditing(s); setValue(''); }}>
                {s.is_set ? 'Обновить' : 'Задать'}
              </Button>
              {s.is_set && (
                <Button variant="destructive" size="sm" onClick={() => setDeleteKind(s.kind)}>
                  Удалить
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      ))}

      <Dialog open={!!editing} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editing ? SECRET_LABELS[editing.kind] : 'Секрет'}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="secret-value">Значение</Label>
            <Input
              id="secret-value"
              type="password"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditing(null)}>Отмена</Button>
            <Button onClick={handleSave} disabled={!value.trim() || setSecret.isPending}>
              Сохранить
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteKind}
        onOpenChange={(open) => !open && setDeleteKind(null)}
        title="Удалить секрет"
        description="Секрет будет удалён. Бот может перестать работать с этим каналом."
        onConfirm={handleDelete}
        loading={deleteSecret.isPending}
      />
    </div>
  );
}

function RagTab() {
  const { data: status, isLoading } = useRagIndexStatus();
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
          {isLoading ? (
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
                    <span className="text-muted-foreground ml-auto">score {r.score.toFixed(3)}</span>
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

export function BotAdminPage() {
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Администрирование бота</h1>
      <Tabs defaultValue="settings">
        <TabsList>
          <TabsTrigger value="settings">Настройки</TabsTrigger>
          <TabsTrigger value="secrets">Секреты</TabsTrigger>
          <TabsTrigger value="rag">RAG</TabsTrigger>
        </TabsList>
        <TabsContent value="settings" className="mt-4">
          <SettingsTab />
        </TabsContent>
        <TabsContent value="secrets" className="mt-4">
          <SecretsTab />
        </TabsContent>
        <TabsContent value="rag" className="mt-4">
          <RagTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
