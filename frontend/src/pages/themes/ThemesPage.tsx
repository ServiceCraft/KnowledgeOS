import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Plus, Pencil, Trash2 } from 'lucide-react';
import { SearchInput } from '@/components/shared/SearchInput';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { EmptyState } from '@/components/shared/EmptyState';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import { useThemesList, useCreateTheme, useUpdateTheme, useDeleteTheme } from '@/hooks/useThemes';
import type { Theme } from '@/types';
import { toast } from 'sonner';

export function ThemesPage() {
  const navigate = useNavigate();
  const [query, setQuery] = useState('');
  const [page] = useState(1);

  const { data, isLoading, isError } = useThemesList({ query: query || undefined, page, limit: 50 });
  const createTheme = useCreateTheme();
  const updateTheme = useUpdateTheme();
  const deleteTheme = useDeleteTheme();

  const [showForm, setShowForm] = useState(false);
  const [editingTheme, setEditingTheme] = useState<Theme | null>(null);
  const [formName, setFormName] = useState('');
  const [formDesc, setFormDesc] = useState('');
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const openCreate = () => {
    setEditingTheme(null);
    setFormName('');
    setFormDesc('');
    setShowForm(true);
  };

  const openEdit = (theme: Theme) => {
    setEditingTheme(theme);
    setFormName(theme.name);
    setFormDesc(theme.description);
    setShowForm(true);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (editingTheme) {
      updateTheme.mutate(
        { id: editingTheme.id, data: { name: formName, description: formDesc } },
        {
          onSuccess: () => { setShowForm(false); toast.success('Тема обновлена'); },
          onError: () => toast.error('Не удалось обновить тему'),
        }
      );
    } else {
      createTheme.mutate(
        { name: formName, description: formDesc },
        {
          onSuccess: () => { setShowForm(false); toast.success('Тема создана'); },
          onError: () => toast.error('Не удалось создать тему'),
        }
      );
    }
  };

  const handleDelete = () => {
    if (!deleteId) return;
    deleteTheme.mutate(deleteId, {
      onSuccess: () => { setDeleteId(null); toast.success('Тема удалена'); },
      onError: () => toast.error('Не удалось удалить тему'),
    });
  };

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message="Не удалось загрузить темы." />;

  const themes = data?.data ?? [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Темы</h1>
        <Button onClick={openCreate}>
          <Plus className="h-4 w-4 mr-2" />
          Добавить тему
        </Button>
      </div>

      <SearchInput onSearch={setQuery} placeholder="Поиск по темам..." className="max-w-sm" />

      {themes.length === 0 ? (
        <EmptyState title="Темы не найдены" message="Темы ещё не созданы." />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {themes.map((theme) => (
            <Card
              key={theme.id}
              className="cursor-pointer hover:border-primary/50 transition-colors"
              onClick={() => navigate(`/kb/qa?theme=${theme.id}`)}
            >
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">{theme.name}</CardTitle>
                  <div className="flex gap-1" onClick={(e) => e.stopPropagation()}>
                    <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => openEdit(theme)}>
                      <Pencil className="h-3 w-3" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-destructive"
                      onClick={() => setDeleteId(theme.id)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <CardDescription className="line-clamp-2">
                  {theme.description || 'Без описания'}
                </CardDescription>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={showForm} onOpenChange={setShowForm}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingTheme ? 'Редактировать тему' : 'Создать тему'}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Название</Label>
              <Input id="name" value={formName} onChange={(e) => setFormName(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Описание</Label>
              <Textarea
                id="description"
                value={formDesc}
                onChange={(e) => setFormDesc(e.target.value)}
                rows={3}
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setShowForm(false)}>
                Отмена
              </Button>
              <Button
                type="submit"
                disabled={createTheme.isPending || updateTheme.isPending}
              >
                {editingTheme ? 'Сохранить' : 'Создать'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Удалить тему"
        description="Вы уверены, что хотите удалить эту тему? QA-пары в этой теме не будут удалены."
        onConfirm={handleDelete}
        loading={deleteTheme.isPending}
      />
    </div>
  );
}
