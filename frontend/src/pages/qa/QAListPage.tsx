import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Plus, ChevronRight, ChevronLeft, Star, ExternalLink } from 'lucide-react';
import { SearchInput } from '@/components/shared/SearchInput';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { EmptyState } from '@/components/shared/EmptyState';
import { useQAList, useCreateQA, useUpdateQA } from '@/hooks/useQA';
import { useThemesList } from '@/hooks/useThemes';
import type { QAPair } from '@/types';
import { toast } from 'sonner';

function FrequencyBadge({ value }: { value: number }) {
  let color = 'bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400';
  if (value >= 10) color = 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400';
  else if (value >= 5) color = 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-400';
  else if (value >= 1) color = 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400';

  return (
    <div className={`shrink-0 rounded-lg px-2.5 py-1.5 flex items-center justify-center font-bold text-lg tabular-nums ${color}`}>
      {value} <span className="text-xs font-normal ml-1 opacity-70">раз</span>
    </div>
  );
}

function QARow({ item, themeName, onNavigate }: { item: QAPair; themeName: string | null; onNavigate: () => void }) {
  const [answer, setAnswer] = useState(item.answer || '');
  const [answerDirty, setAnswerDirty] = useState(false);
  const updateQA = useUpdateQA();

  useEffect(() => {
    setAnswer(item.answer || '');
    setAnswerDirty(false);
  }, [item.answer]);

  const saveAnswer = () => {
    updateQA.mutate(
      { id: item.id, data: { answer } },
      {
        onSuccess: () => { setAnswerDirty(false); toast.success('Ответ сохранён'); },
        onError: () => toast.error('Не удалось сохранить'),
      }
    );
  };

  return (
    <div className="rounded-lg border bg-card p-4 space-y-2">
      {/* Top row: frequency + question + meta + link */}
      <div className="flex items-center gap-3">
        <FrequencyBadge value={item.frequency} />
        <div className="flex-1 min-w-0">
          <p className="font-medium text-sm leading-snug">{item.question}</p>
          <div className="flex items-center gap-2 mt-1">
            {item.is_faq && (
              <Badge variant="secondary" className="text-xs gap-1 px-1.5 py-0">
                <Star className="h-3 w-3" />
                FAQ
              </Badge>
            )}
            {item.is_locked && (
              <Badge variant="outline" className="text-xs px-1.5 py-0">Заблокирован</Badge>
            )}
            {themeName && (
              <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded">
                {themeName}
              </span>
            )}
            <span className="text-xs text-muted-foreground ml-auto">
              {new Date(item.updated_at).toLocaleDateString()}
            </span>
          </div>
        </div>
        <Button
          variant="ghost"
          size="icon"
          className="shrink-0 h-8 w-8"
          onClick={onNavigate}
          title="Открыть детали"
        >
          <ExternalLink className="h-4 w-4 text-muted-foreground" />
        </Button>
      </div>

      {/* Answer inline edit */}
      <div className="flex gap-2 items-start">
        <Textarea
          value={answer}
          onChange={(e) => { setAnswer(e.target.value); setAnswerDirty(true); }}
          placeholder="Введите ответ..."
          rows={2}
          className="text-sm flex-1 min-h-[2.5rem] resize-y"
        />
        {answerDirty && (
          <Button size="sm" className="shrink-0 h-9 px-4 bg-green-600 hover:bg-green-700 text-white" onClick={saveAnswer} disabled={updateQA.isPending}>
            {updateQA.isPending ? 'Сохранение...' : 'Сохранить'}
          </Button>
        )}
      </div>
    </div>
  );
}

export function QAListPage() {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState('');
  const [themeId, setThemeId] = useState<string>('');
  const [isFaq, setIsFaq] = useState<string>('');
  const [sort, setSort] = useState<string>('-frequency');
  const [showCreate, setShowCreate] = useState(false);
  const [newQuestion, setNewQuestion] = useState('');
  const [newAnswer, setNewAnswer] = useState('');
  const [newThemeId, setNewThemeId] = useState<string>('');

  const limit = 20;
  const { data, isLoading, isError } = useQAList({
    query: query || undefined,
    theme_id: themeId || undefined,
    is_faq: isFaq === 'true' ? true : isFaq === 'false' ? false : undefined,
    sort: sort || undefined,
    page,
    limit,
  });

  const { data: themesData } = useThemesList({ limit: 100 });
  const createQA = useCreateQA();

  const themes = themesData?.data ?? [];
  const items = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / limit) || 1;

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    createQA.mutate(
      {
        question: newQuestion,
        answer: newAnswer || undefined,
        theme_id: newThemeId || undefined,
      },
      {
        onSuccess: () => {
          setShowCreate(false);
          setNewQuestion('');
          setNewAnswer('');
          setNewThemeId('');
          toast.success('Вопрос создан');
        },
        onError: () => toast.error('Не удалось создать вопрос'),
      }
    );
  };

  const getThemeName = (id?: string) => {
    if (!id) return null;
    return themes.find((t) => t.id === id)?.name ?? null;
  };

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message="Не удалось загрузить пары Q&A." />;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Вопросы и ответы</h1>
          <p className="text-sm text-muted-foreground mt-1">{total} записей</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Добавить Q&A
        </Button>
      </div>

      <div className="flex items-center gap-3 flex-wrap">
        <SearchInput
          onSearch={(v) => { setQuery(v); setPage(1); }}
          placeholder="Поиск по вопросам..."
          className="w-72"
        />
        <Select value={themeId} onValueChange={(v) => { setThemeId(v ?? ''); setPage(1); }}>
          <SelectTrigger className="w-48">
            <SelectValue placeholder="Все темы">
              {themeId ? themes.find((t) => t.id === themeId)?.name ?? 'Все темы' : 'Все темы'}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">Все темы</SelectItem>
            {themes.map((t) => (
              <SelectItem key={t.id} value={t.id}>
                {t.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={isFaq} onValueChange={(v) => { setIsFaq(v ?? ''); setPage(1); }}>
          <SelectTrigger className="w-36">
            <SelectValue placeholder="Все">
              {isFaq === 'true' ? 'Только FAQ' : isFaq === 'false' ? 'Не FAQ' : 'Все'}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">Все</SelectItem>
            <SelectItem value="true">Только FAQ</SelectItem>
            <SelectItem value="false">Не FAQ</SelectItem>
          </SelectContent>
        </Select>
        <Select value={sort} onValueChange={(v) => { setSort(v ?? ''); setPage(1); }}>
          <SelectTrigger className="w-52">
            <SelectValue placeholder="Сортировка">
              {sort === '-frequency' ? 'Частота: по убыванию' : sort === 'frequency' ? 'Частота: по возрастанию' : sort === '-created_at' ? 'Дата: сначала новые' : sort === 'created_at' ? 'Дата: сначала старые' : 'Сортировка'}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="-frequency">Частота: по убыванию</SelectItem>
            <SelectItem value="frequency">Частота: по возрастанию</SelectItem>
            <SelectItem value="-created_at">Дата: сначала новые</SelectItem>
            <SelectItem value="created_at">Дата: сначала старые</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {items.length === 0 ? (
        <EmptyState title="Пары Q&A не найдены" message="Попробуйте изменить фильтры или создайте новую пару." />
      ) : (
        <div className="space-y-3">
          {items.map((item: QAPair) => (
            <QARow
              key={item.id}
              item={item}
              themeName={getThemeName(item.theme_id)}
              onNavigate={() => navigate(`/kb/qa/${item.id}`)}
            />
          ))}
        </div>
      )}

      {total > limit && (
        <div className="flex items-center justify-between pt-2">
          <p className="text-sm text-muted-foreground">
            {(page - 1) * limit + 1}–{Math.min(page * limit, total)} из {total}
          </p>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => setPage(page - 1)} disabled={page <= 1}>
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="text-sm tabular-nums">{page} / {totalPages}</span>
            <Button variant="outline" size="sm" onClick={() => setPage(page + 1)} disabled={page >= totalPages}>
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Создать вопрос</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="question">Вопрос</Label>
              <Input
                id="question"
                value={newQuestion}
                onChange={(e) => setNewQuestion(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="answer">Ответ <span className="text-muted-foreground font-normal">(необязательно)</span></Label>
              <Textarea
                id="answer"
                value={newAnswer}
                onChange={(e) => setNewAnswer(e.target.value)}
                rows={4}
                placeholder="Можно добавить позже из списка"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="theme">Тема</Label>
              <Select value={newThemeId} onValueChange={(v) => setNewThemeId(v ?? '')}>
                <SelectTrigger>
                  <SelectValue placeholder="Без темы">
                    {newThemeId ? themes.find((t) => t.id === newThemeId)?.name ?? 'Без темы' : 'Без темы'}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">Без темы</SelectItem>
                  {themes.map((t) => (
                    <SelectItem key={t.id} value={t.id}>
                      {t.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>
                Отмена
              </Button>
              <Button type="submit" disabled={createQA.isPending}>
                {createQA.isPending ? 'Создание...' : 'Создать'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
