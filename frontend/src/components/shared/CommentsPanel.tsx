import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { MessageSquare, Pencil, Trash2, Send } from 'lucide-react';
import { useCommentsList, useCreateComment, useUpdateComment, useDeleteComment } from '@/hooks/useComments';
import { useAuthStore } from '@/stores/authStore';
import { LoadingState } from './LoadingState';
import { ConfirmDialog } from './ConfirmDialog';
import { toast } from 'sonner';

interface CommentsPanelProps {
  entityType: string;
  entityId: string;
}

export function CommentsPanel({ entityType, entityId }: CommentsPanelProps) {
  const { data, isLoading } = useCommentsList(entityType, entityId);
  const createComment = useCreateComment(entityType, entityId);
  const updateComment = useUpdateComment(entityType, entityId);
  const deleteComment = useDeleteComment(entityType, entityId);
  const user = useAuthStore((s) => s.user);

  const [newBody, setNewBody] = useState('');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editBody, setEditBody] = useState('');
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const handleCreate = () => {
    if (!newBody.trim()) return;
    createComment.mutate(
      { body: newBody },
      {
        onSuccess: () => {
          setNewBody('');
          toast.success('Комментарий добавлен');
        },
        onError: () => toast.error('Не удалось добавить комментарий'),
      }
    );
  };

  const handleUpdate = () => {
    if (!editingId || !editBody.trim()) return;
    updateComment.mutate(
      { commentId: editingId, body: editBody },
      {
        onSuccess: () => {
          setEditingId(null);
          setEditBody('');
          toast.success('Комментарий обновлён');
        },
        onError: () => toast.error('Не удалось обновить комментарий'),
      }
    );
  };

  const handleDelete = () => {
    if (!deleteId) return;
    deleteComment.mutate(deleteId, {
      onSuccess: () => {
        setDeleteId(null);
        toast.success('Комментарий удалён');
      },
      onError: () => toast.error('Не удалось удалить комментарий'),
    });
  };

  if (isLoading) return <LoadingState />;

  const comments = data?.data ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <MessageSquare className="h-4 w-4" />
          Комментарии ({data?.total ?? 0})
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-3">
          {comments.map((comment) => (
            <div key={comment.id} className="space-y-2">
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  {editingId === comment.id ? (
                    <div className="space-y-2">
                      <Textarea
                        value={editBody}
                        onChange={(e) => setEditBody(e.target.value)}
                        rows={2}
                      />
                      <div className="flex gap-2">
                        <Button size="sm" onClick={handleUpdate} disabled={updateComment.isPending}>
                          Сохранить
                        </Button>
                        <Button size="sm" variant="outline" onClick={() => setEditingId(null)}>
                          Отмена
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <p className="text-sm">{comment.body}</p>
                  )}
                </div>
                {user && comment.author_id === user.id && editingId !== comment.id && (
                  <div className="flex gap-1 ml-2">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      onClick={() => {
                        setEditingId(comment.id);
                        setEditBody(comment.body);
                      }}
                    >
                      <Pencil className="h-3 w-3" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-destructive"
                      onClick={() => setDeleteId(comment.id)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </div>
                )}
              </div>
              <p className="text-xs text-muted-foreground">
                {new Date(comment.created_at).toLocaleString()}
              </p>
              <Separator />
            </div>
          ))}
        </div>

        <div className="flex gap-2">
          <Textarea
            value={newBody}
            onChange={(e) => setNewBody(e.target.value)}
            placeholder="Добавить комментарий..."
            rows={2}
            className="flex-1"
          />
          <Button
            size="icon"
            onClick={handleCreate}
            disabled={!newBody.trim() || createComment.isPending}
          >
            <Send className="h-4 w-4" />
          </Button>
        </div>

        <ConfirmDialog
          open={!!deleteId}
          onOpenChange={(open) => !open && setDeleteId(null)}
          title="Удалить комментарий"
          description="Вы уверены, что хотите удалить этот комментарий? Это действие нельзя отменить."
          onConfirm={handleDelete}
          loading={deleteComment.isPending}
        />
      </CardContent>
    </Card>
  );
}
