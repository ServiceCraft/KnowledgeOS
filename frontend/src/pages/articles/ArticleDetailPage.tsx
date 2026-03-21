import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent } from '@/components/ui/card';
import { ArrowLeft, Pencil, Trash2, Save, X } from 'lucide-react';
import { useArticleDetail, useUpdateArticle, useDeleteArticle } from '@/hooks/useArticles';
import { MarkdownEditor } from '@/components/shared/MarkdownEditor';
import { MarkdownViewer } from '@/components/shared/MarkdownViewer';
import { CommentsPanel } from '@/components/shared/CommentsPanel';
import { LinksPanel } from '@/components/shared/LinksPanel';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import { toast } from 'sonner';

export function ArticleDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: article, isLoading, isError } = useArticleDetail(id!);
  const updateArticle = useUpdateArticle();
  const deleteArticle = useDeleteArticle();

  const [editing, setEditing] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');

  const startEditing = () => {
    if (!article) return;
    setTitle(article.title);
    setBody(article.body);
    setEditing(true);
  };

  const handleSave = () => {
    updateArticle.mutate(
      { id: id!, data: { title, body } },
      {
        onSuccess: () => {
          setEditing(false);
          toast.success('Статья обновлена');
        },
        onError: () => toast.error('Не удалось обновить статью'),
      }
    );
  };

  const handleDelete = () => {
    deleteArticle.mutate(id!, {
      onSuccess: () => {
        toast.success('Статья удалена');
        navigate('/kb/articles');
      },
      onError: () => toast.error('Не удалось удалить статью'),
    });
  };

  if (isLoading) return <LoadingState />;
  if (isError || !article) return <ErrorState message="Статья не найдена." />;

  return (
    <div className="space-y-6 max-w-4xl">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate('/kb/articles')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-2xl font-semibold flex-1">
          {editing ? (
            <Input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="text-2xl font-semibold h-auto"
            />
          ) : (
            article.title
          )}
        </h1>
        {!editing && (
          <div className="flex gap-2">
            <Button variant="outline" onClick={startEditing}>
              <Pencil className="h-4 w-4 mr-2" />
              Редактировать
            </Button>
            <Button variant="destructive" onClick={() => setShowDelete(true)}>
              <Trash2 className="h-4 w-4 mr-2" />
              Удалить
            </Button>
          </div>
        )}
        {editing && (
          <div className="flex gap-2">
            <Button onClick={handleSave} disabled={updateArticle.isPending}>
              <Save className="h-4 w-4 mr-2" />
              {updateArticle.isPending ? 'Сохранение...' : 'Сохранить'}
            </Button>
            <Button variant="outline" onClick={() => setEditing(false)}>
              <X className="h-4 w-4 mr-2" />
              Отмена
            </Button>
          </div>
        )}
      </div>

      <Card>
        <CardContent className="pt-6">
          {editing ? (
            <MarkdownEditor value={body} onChange={setBody} />
          ) : (
            <MarkdownViewer source={article.body || '*Контент отсутствует.*'} />
          )}
        </CardContent>
      </Card>

      <p className="text-sm text-muted-foreground">
        Последнее обновление: {new Date(article.updated_at).toLocaleString()}
      </p>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <CommentsPanel entityType="articles" entityId={id!} />
        <LinksPanel entityType="articles" entityId={id!} />
      </div>

      <ConfirmDialog
        open={showDelete}
        onOpenChange={setShowDelete}
        title="Удалить статью"
        description="Вы уверены, что хотите удалить эту статью? Это действие нельзя отменить."
        onConfirm={handleDelete}
        loading={deleteArticle.isPending}
      />
    </div>
  );
}
