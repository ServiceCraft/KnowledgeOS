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
import { Plus, ChevronRight, ChevronLeft, Star, ExternalLink, Bot, Check, X, Pencil } from 'lucide-react';
import { SearchInput } from '@/components/shared/SearchInput';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { EmptyState } from '@/components/shared/EmptyState';
import { useQAList, useCreateQA, useUpdateQA, useReviewAIAnswer } from '@/hooks/useQA';
import { useThemesList } from '@/hooks/useThemes';
import { usePermissions } from '@/hooks/usePermissions';
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

function AIStatusBadge({ status }: { status: string }) {
  switch (status) {
    case 'accepted':
      return <Badge className="text-xs gap-1 px-1.5 py-0 bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400"><Check className="h-3 w-3" />AI принят</Badge>;
    case 'rejected':
      return <Badge className="text-xs gap-1 px-1.5 py-0 bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400"><X className="h-3 w-3" />AI отклонён</Badge>;
    case 'edited':
      return <Badge className="text-xs gap-1 px-1.5 py-0 bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400"><Pencil className="h-3 w-3" />AI исправлен</Badge>;
    default:
      return null;
  }
}

function QARow({ item, themeName, onNavigate, canWrite }: { item: QAPair; themeName: string | null; onNavigate: () => void; canWrite: boolean }) {
  const [answer, setAnswer] = useState(item.answer || '');
  const [answerDirty, setAnswerDirty] = useState(false);
  const [editingAI, setEditingAI] = useState(false);
  const [editedAIAnswer, setEditedAIAnswer] = useState('');
  const updateQA = useUpdateQA();
  const reviewAI = useReviewAIAnswer();

  useEffect(() => {
    setAnswer(item.answer || '');
    setAnswerDirty(false);
  }, [item.answer]);

  useEffect(() => {
    if (item.ai_answer) {
      setEditedAIAnswer(item.ai_answer);
    }
  }, [item.ai_answer]);

  const saveAnswer = () => {
    updateQA.mutate(
      { id: item.id, data: { answer } },
      {
        onSuccess: () => { setAnswerDirty(false); toast.success('Ответ сохранён'); },
        onError: () => toast.error('Не удалось сохранить'),
      }
    );
  };

  const handleAccept = () => {
    reviewAI.mutate(
      { id: item.id, data: { action: 'accept' } },
      {
        onSuccess: () => toast.success('AI-ответ принят'),
        onError: () => toast.error('Не удалось принять'),
      }
    );
  };

  const handleReject = () => {
    reviewAI.mutate(
      { id: item.id, data: { action: 'reject' } },
      {
        onSuccess: () => toast.success('AI-ответ отклонён'),
        onError: () => toast.error('Не удалось отклонить'),
      }
    );
  };

  const handleEditSave = () => {
    reviewAI.mutate(
      { id: item.id, data: { action: 'edit', edited_answer: editedAIAnswer } },
      {
        onSuccess: () => { setEditingAI(false); toast.success('AI-ответ отредактирован и принят'); },
        onError: () => toast.error('Не удалось сохранить'),
      }
    );
  };

  const isPending = item.ai_status === 'pending';

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
            {item.ai_status && item.ai_status !== 'pending' && (
              <AIStatusBadge status={item.ai_status} />
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

      {/* AI review section */}
      {isPending && canWrite && (
        <div className="rounded-md border border-violet-200 bg-violet-50 dark:border-violet-800 dark:bg-violet-950/30 p-3 space-y-2">
          <div className="flex items-center gap-2">
            <Bot className="h-4 w-4 text-violet-600 dark:text-violet-400" />
            <span className="text-xs font-medium text-violet-700 dark:text-violet-300">AI-предложенный ответ</span>
          </div>
          {item.answer && (
            <div className="text-xs text-muted-foreground bg-muted/50 rounded p-2">
              <span className="font-medium">Текущий ответ: </span>{item.answer}
            </div>
          )}
          {editingAI ? (
            <div className="space-y-2">
              <Textarea
                value={editedAIAnswer}
                onChange={(e) => setEditedAIAnswer(e.target.value)}
                rows={3}
                className="text-sm resize-y"
              />
              <div className="flex gap-2">
                <Button size="sm" onClick={handleEditSave} disabled={reviewAI.isPending} className="bg-green-600 hover:bg-green-700 text-white">
                  {reviewAI.isPending ? 'Сохранение...' : 'Сохранить'}
                </Button>
                <Button size="sm" variant="outline" onClick={() => { setEditingAI(false); setEditedAIAnswer(item.ai_answer || ''); }}>
                  Отмена
                </Button>
              </div>
            </div>
          ) : (
            <>
              <p className="text-sm whitespace-pre-wrap">{item.ai_answer}</p>
              <div className="flex gap-2 pt-1">
                <Button size="sm" variant="outline" className="text-red-600 border-red-200 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950/30" onClick={handleReject} disabled={reviewAI.isPending}>
                  <X className="h-3.5 w-3.5 mr-1" />
                  Отклонить
                </Button>
                <Button size="sm" className="bg-green-600 hover:bg-green-700 text-white" onClick={handleAccept} disabled={reviewAI.isPending}>
                  <Check className="h-3.5 w-3.5 mr-1" />
                  Принять
                </Button>
                <Button size="sm" variant="outline" className="text-yellow-600 border-yellow-200 hover:bg-yellow-50 dark:border-yellow-800 dark:hover:bg-yellow-950/30" onClick={() => setEditingAI(true)} disabled={reviewAI.isPending}>
                  <Pencil className="h-3.5 w-3.5 mr-1" />
                  Править
                </Button>
              </div>
            </>
          )}
        </div>
      )}

      {/* Answer inline edit (hidden when AI pending). Read-only for operators. */}
      {(!isPending || !canWrite) &&
        (canWrite ? (
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
        ) : (
          <div className="text-sm whitespace-pre-wrap">
            {item.answer || <span className="text-muted-foreground">Ответ не задан</span>}
          </div>
        ))}
    </div>
  );
}

export function QAListPage() {
  const navigate = useNavigate();
  const { canWrite } = usePermissions();
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState('');
  const [themeId, setThemeId] = useState<string>('');
  const [isFaq, setIsFaq] = useState<string>('');
  const [sort, setSort] = useState<string>('-frequency');
  const [aiStatus, setAiStatus] = useState<string>('');
  const [showCreate, setShowCreate] = useState(false);
  const [newQuestion, setNewQuestion] = useState('');
  const [newAnswer, setNewAnswer] = useState('');
  const [newThemeId, setNewThemeId] = useState<string>('');

  const limit = 20;
  const { data, isLoading, isError } = useQAList({
    query: query || undefined,
    theme_id: themeId || undefined,
    is_faq: isFaq === 'true' ? true : isFaq === 'false' ? false : undefined,
    ai_status: aiStatus || undefined,
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
        {canWrite && (
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Добавить Q&A
          </Button>
        )}
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
        <Select value={aiStatus} onValueChange={(v) => { setAiStatus(v ?? ''); setPage(1); }}>
          <SelectTrigger className="w-48">
            <SelectValue placeholder="AI статус">
              {aiStatus === 'pending' ? 'Ожидает проверки' : aiStatus === 'accepted' ? 'AI принят' : aiStatus === 'rejected' ? 'AI отклонён' : aiStatus === 'edited' ? 'AI исправлен' : 'AI статус'}
            </SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="">Все</SelectItem>
            <SelectItem value="pending">Ожидает проверки</SelectItem>
            <SelectItem value="accepted">AI принят</SelectItem>
            <SelectItem value="rejected">AI отклонён</SelectItem>
            <SelectItem value="edited">AI исправлен</SelectItem>
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
              canWrite={canWrite}
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
