import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Link2, Trash2, Plus, ExternalLink } from 'lucide-react';
import { useLinksList, useCreateLink, useDeleteLink } from '@/hooks/useLinks';
import { LoadingState } from './LoadingState';
import { ConfirmDialog } from './ConfirmDialog';
import { toast } from 'sonner';

interface LinksPanelProps {
  entityType: string;
  entityId: string;
}

export function LinksPanel({ entityType, entityId }: LinksPanelProps) {
  const { data, isLoading } = useLinksList(entityType, entityId);
  const createLink = useCreateLink(entityType, entityId);
  const deleteLink = useDeleteLink(entityType, entityId);

  const [showForm, setShowForm] = useState(false);
  const [url, setUrl] = useState('');
  const [label, setLabel] = useState('');
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const handleCreate = () => {
    if (!url.trim() && !label.trim()) return;
    createLink.mutate(
      { url: url || undefined, label: label || undefined },
      {
        onSuccess: () => {
          setUrl('');
          setLabel('');
          setShowForm(false);
          toast.success('Ссылка добавлена');
        },
        onError: () => toast.error('Не удалось добавить ссылку'),
      }
    );
  };

  const handleDelete = () => {
    if (!deleteId) return;
    deleteLink.mutate(deleteId, {
      onSuccess: () => {
        setDeleteId(null);
        toast.success('Ссылка удалена');
      },
      onError: () => toast.error('Не удалось удалить ссылку'),
    });
  };

  if (isLoading) return <LoadingState />;

  const links = data?.data ?? [];

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2 text-base">
            <Link2 className="h-4 w-4" />
            Ссылки ({data?.total ?? 0})
          </CardTitle>
          <Button variant="ghost" size="sm" onClick={() => setShowForm(!showForm)}>
            <Plus className="h-4 w-4 mr-1" />
            Добавить
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {showForm && (
          <div className="space-y-2 p-3 border rounded-md">
            <Input
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="Название"
            />
            <Input
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="URL (необязательно для внешних ссылок)"
            />
            <div className="flex gap-2">
              <Button size="sm" onClick={handleCreate} disabled={createLink.isPending}>
                Сохранить
              </Button>
              <Button size="sm" variant="outline" onClick={() => setShowForm(false)}>
                Отмена
              </Button>
            </div>
          </div>
        )}

        {links.map((link) => (
          <div key={link.id} className="flex items-center justify-between p-2 border rounded-md">
            <div className="flex items-center gap-2 min-w-0">
              {link.url ? (
                <a
                  href={link.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-primary hover:underline flex items-center gap-1 truncate"
                >
                  <ExternalLink className="h-3 w-3 shrink-0" />
                  {link.label || link.url}
                </a>
              ) : (
                <span className="text-sm truncate">{link.label || 'Внутренняя ссылка'}</span>
              )}
            </div>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-destructive shrink-0"
              onClick={() => setDeleteId(link.id)}
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
        ))}

        <ConfirmDialog
          open={!!deleteId}
          onOpenChange={(open) => !open && setDeleteId(null)}
          title="Удалить ссылку"
          description="Вы уверены, что хотите удалить эту ссылку?"
          onConfirm={handleDelete}
          loading={deleteLink.isPending}
        />
      </CardContent>
    </Card>
  );
}
