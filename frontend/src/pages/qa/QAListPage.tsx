import { useState } from 'react';
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
import { Plus, ChevronRight, ChevronLeft, MessageSquareQuote, Star } from 'lucide-react';
import { SearchInput } from '@/components/shared/SearchInput';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { EmptyState } from '@/components/shared/EmptyState';
import { useQAList, useCreateQA } from '@/hooks/useQA';
import { useThemesList } from '@/hooks/useThemes';
import type { QAPair } from '@/types';
import { toast } from 'sonner';

export function QAListPage() {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState('');
  const [themeId, setThemeId] = useState<string>('');
  const [isFaq, setIsFaq] = useState<string>('');
  const [showCreate, setShowCreate] = useState(false);
  const [newQuestion, setNewQuestion] = useState('');
  const [newAnswer, setNewAnswer] = useState('');
  const [newThemeId, setNewThemeId] = useState<string>('');

  const limit = 20;
  const { data, isLoading, isError } = useQAList({
    query: query || undefined,
    theme_id: themeId || undefined,
    is_faq: isFaq === 'true' ? true : isFaq === 'false' ? false : undefined,
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
        answer: newAnswer,
        theme_id: newThemeId || undefined,
      },
      {
        onSuccess: () => {
          setShowCreate(false);
          setNewQuestion('');
          setNewAnswer('');
          setNewThemeId('');
          toast.success('Пара Q&A создана');
        },
        onError: () => toast.error('Не удалось создать пару Q&A'),
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
      </div>

      {items.length === 0 ? (
        <EmptyState title="Пары Q&A не найдены" message="Попробуйте изменить фильтры или создайте новую пару." />
      ) : (
        <div className="space-y-2">
          {items.map((item: QAPair) => {
            const themeName = getThemeName(item.theme_id);
            return (
              <div
                key={item.id}
                onClick={() => navigate(`/kb/qa/${item.id}`)}
                className="group flex items-start gap-4 p-4 rounded-lg border bg-card hover:bg-accent/50 hover:border-primary/30 transition-all cursor-pointer"
              >
                <div className="mt-0.5 shrink-0">
                  <div className="h-9 w-9 rounded-lg bg-primary/10 flex items-center justify-center">
                    <MessageSquareQuote className="h-4 w-4 text-primary" />
                  </div>
                </div>
                <div className="flex-1 min-w-0 space-y-1">
                  <p className="font-medium text-sm leading-snug line-clamp-1">{item.question}</p>
                  <p className="text-sm text-muted-foreground leading-relaxed line-clamp-2">{item.answer}</p>
                  <div className="flex items-center gap-2 pt-1">
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
                <ChevronRight className="h-4 w-4 text-muted-foreground mt-2 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity" />
              </div>
            );
          })}
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
            <DialogTitle>Создать пару Q&A</DialogTitle>
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
              <Label htmlFor="answer">Ответ</Label>
              <Textarea
                id="answer"
                value={newAnswer}
                onChange={(e) => setNewAnswer(e.target.value)}
                rows={4}
                required
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
