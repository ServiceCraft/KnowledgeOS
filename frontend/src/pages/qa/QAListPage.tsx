import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Button } from '@/components/ui/button';
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
import { Plus } from 'lucide-react';
import { SearchInput } from '@/components/shared/SearchInput';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { EmptyState } from '@/components/shared/EmptyState';
import { PaginationBar } from '@/components/shared/PaginationBar';
import { QARow } from '@/components/qa/QARow';
import { useQAList, useCreateQA, useUpdateQA, useReviewAIAnswer } from '@/hooks/useQA';
import { useThemesList } from '@/hooks/useThemes';
import { usePermissions } from '@/hooks/usePermissions';
import type { QAPair } from '@/types';
import { toast } from 'sonner';

export function QAListPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { canWrite } = usePermissions();
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState('');
  const [themeId, setThemeId] = useState<string>(() => searchParams.get('theme') ?? '');
  const [isFaq, setIsFaq] = useState<string>('');
  const [sort, setSort] = useState<string>('-frequency');
  const [aiStatus, setAiStatus] = useState<string>('');
  const [showCreate, setShowCreate] = useState(false);
  const [newQuestion, setNewQuestion] = useState('');
  const [newAnswer, setNewAnswer] = useState('');
  const [newThemeId, setNewThemeId] = useState<string>('');

  useEffect(() => {
    const themeFromUrl = searchParams.get('theme');
    if (themeFromUrl) {
      setThemeId(themeFromUrl);
      setPage(1);
    }
  }, [searchParams]);

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
  const updateQA = useUpdateQA();
  const reviewAI = useReviewAIAnswer();

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

  const handleSaveAnswer = (id: string, answer: string) => {
    updateQA.mutate(
      { id, data: { answer } },
      {
        onSuccess: () => toast.success('Ответ сохранён'),
        onError: () => toast.error('Не удалось сохранить'),
      }
    );
  };

  const handleReviewAI = (id: string, data: Parameters<typeof reviewAI.mutate>[0]['data']) => {
    reviewAI.mutate(
      { id, data },
      {
        onSuccess: () => {
          if (data.action === 'accept') toast.success('AI-ответ принят');
          else if (data.action === 'reject') toast.success('AI-ответ отклонён');
          else toast.success('AI-ответ отредактирован и принят');
        },
        onError: () => toast.error('Не удалось выполнить действие'),
      }
    );
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
              onSaveAnswer={handleSaveAnswer}
              onReviewAI={handleReviewAI}
              isSavingAnswer={updateQA.isPending}
              isReviewing={reviewAI.isPending}
            />
          ))}
        </div>
      )}

      {total > limit && (
        <div className="flex items-center justify-between pt-2">
          <p className="text-sm text-muted-foreground">
            {(page - 1) * limit + 1}–{Math.min(page * limit, total)} из {total}
          </p>
          <PaginationBar page={page} totalPages={totalPages} onPageChange={setPage} />
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
