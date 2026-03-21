import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Plus, FileText, ChevronLeft, ChevronRight } from 'lucide-react';
import { SearchInput } from '@/components/shared/SearchInput';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { EmptyState } from '@/components/shared/EmptyState';
import { useArticlesList, useCreateArticle } from '@/hooks/useArticles';
import { toast } from 'sonner';

export function ArticleListPage() {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState('');
  const [showCreate, setShowCreate] = useState(false);
  const [newTitle, setNewTitle] = useState('');

  const limit = 20;
  const { data, isLoading, isError } = useArticlesList({
    query: query || undefined,
    page,
    limit,
  });
  const createArticle = useCreateArticle();

  const items = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / limit) || 1;

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    createArticle.mutate(
      { title: newTitle, body: '' },
      {
        onSuccess: (article) => {
          setShowCreate(false);
          setNewTitle('');
          toast.success('Статья создана');
          navigate(`/kb/articles/${article.id}`);
        },
        onError: () => toast.error('Не удалось создать статью'),
      }
    );
  };

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message="Не удалось загрузить статьи." />;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Статьи</h1>
          <p className="text-sm text-muted-foreground mt-1">{total} {total === 1 ? 'статья' : total >= 2 && total <= 4 ? 'статьи' : 'статей'}</p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Новая статья
        </Button>
      </div>

      <SearchInput
        onSearch={(v) => { setQuery(v); setPage(1); }}
        placeholder="Поиск по статьям..."
        className="max-w-sm"
      />

      {items.length === 0 ? (
        <EmptyState title="Статьи не найдены" message="Попробуйте изменить поиск или создайте новую статью." />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {items.map((item) => (
            <div
              key={item.id}
              onClick={() => navigate(`/kb/articles/${item.id}`)}
              className="group flex flex-col rounded-lg border bg-card p-4 hover:bg-accent/50 hover:border-primary/30 transition-all cursor-pointer"
            >
              <div className="flex items-start gap-3 mb-3">
                <div className="h-9 w-9 rounded-lg bg-amber-500/10 flex items-center justify-center shrink-0">
                  <FileText className="h-4 w-4 text-amber-600" />
                </div>
                <h3 className="font-medium text-sm leading-snug line-clamp-2 pt-1">{item.title}</h3>
              </div>
              <p className="text-sm text-muted-foreground line-clamp-3 flex-1">
                {item.body?.slice(0, 160) || 'Пустая статья'}
              </p>
              <p className="text-xs text-muted-foreground mt-3 pt-3 border-t">
                {new Date(item.updated_at).toLocaleDateString()}
              </p>
            </div>
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
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Создать статью</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="title">Заголовок</Label>
              <Input
                id="title"
                value={newTitle}
                onChange={(e) => setNewTitle(e.target.value)}
                required
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>
                Отмена
              </Button>
              <Button type="submit" disabled={createArticle.isPending}>
                Создать
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
