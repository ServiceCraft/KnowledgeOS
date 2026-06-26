import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Link2, Trash2, Plus, ExternalLink } from 'lucide-react';
import { useLinksList, useCreateLink, useDeleteLink } from '@/hooks/useLinks';
import { usePermissions } from '@/hooks/usePermissions';
import { LoadingState } from './LoadingState';
import { ConfirmDialog } from './ConfirmDialog';
import { toast } from 'sonner';

interface LinksPanelProps {
  entityType: string;
  entityId: string;
}

type LinkMode = 'external' | 'internal';

const ENTITY_TYPE_LABELS: Record<string, string> = {
  qa: 'Q&A',
  article: 'Статья',
  pricing: 'Прайс',
};

export function LinksPanel({ entityType, entityId }: LinksPanelProps) {
  const { canWrite } = usePermissions();
  const { data, isLoading } = useLinksList(entityType, entityId);
  const createLink = useCreateLink(entityType, entityId);
  const deleteLink = useDeleteLink(entityType, entityId);

  const [showForm, setShowForm] = useState(false);
  const [linkMode, setLinkMode] = useState<LinkMode>('external');
  const [url, setUrl] = useState('');
  const [label, setLabel] = useState('');
  const [targetType, setTargetType] = useState('qa');
  const [targetId, setTargetId] = useState('');
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const resetForm = () => {
    setUrl('');
    setLabel('');
    setTargetId('');
    setLinkMode('external');
    setShowForm(false);
  };

  const handleCreate = () => {
    if (linkMode === 'external') {
      if (!url.trim() && !label.trim()) return;
      createLink.mutate(
        { url: url || undefined, label: label || undefined },
        {
          onSuccess: () => {
            resetForm();
            toast.success('Ссылка добавлена');
          },
          onError: () => toast.error('Не удалось добавить ссылку'),
        }
      );
      return;
    }

    if (!targetId.trim()) {
      toast.error('Укажите ID целевой сущности');
      return;
    }
    createLink.mutate(
      {
        label: label || undefined,
        target_type: targetType,
        target_id: targetId.trim(),
      },
      {
        onSuccess: () => {
          resetForm();
          toast.success('Связь добавлена');
        },
        onError: () => toast.error('Не удалось добавить связь'),
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
          {canWrite && (
            <Button variant="ghost" size="sm" onClick={() => setShowForm(!showForm)}>
              <Plus className="h-4 w-4 mr-1" />
              Добавить
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {showForm && (
          <div className="space-y-2 p-3 border rounded-md">
            <Select value={linkMode} onValueChange={(v) => setLinkMode(v as LinkMode)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="external">Внешняя ссылка</SelectItem>
                <SelectItem value="internal">Сущность БЗ</SelectItem>
              </SelectContent>
            </Select>
            <Input
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder="Название"
            />
            {linkMode === 'external' ? (
              <Input
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://..."
              />
            ) : (
              <>
                <Select value={targetType} onValueChange={(v) => v && setTargetType(v)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="qa">Q&A</SelectItem>
                    <SelectItem value="article">Статья</SelectItem>
                    <SelectItem value="pricing">Прайс</SelectItem>
                  </SelectContent>
                </Select>
                <Input
                  value={targetId}
                  onChange={(e) => setTargetId(e.target.value)}
                  placeholder="UUID сущности"
                />
              </>
            )}
            <div className="flex gap-2">
              <Button size="sm" onClick={handleCreate} disabled={createLink.isPending}>
                Сохранить
              </Button>
              <Button size="sm" variant="outline" onClick={resetForm}>
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
              ) : link.target_type && link.target_id ? (
                <span className="text-sm truncate">
                  {link.label || `${ENTITY_TYPE_LABELS[link.target_type] ?? link.target_type}: ${link.target_id}`}
                </span>
              ) : (
                <span className="text-sm truncate">{link.label || 'Ссылка'}</span>
              )}
            </div>
            {canWrite && (
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 text-destructive shrink-0"
                onClick={() => setDeleteId(link.id)}
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            )}
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
