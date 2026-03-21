import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';

import { ArrowLeft, Pencil, Trash2, Save, X } from 'lucide-react';
import { useQADetail, useUpdateQA, useDeleteQA } from '@/hooks/useQA';
import { CommentsPanel } from '@/components/shared/CommentsPanel';
import { LinksPanel } from '@/components/shared/LinksPanel';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import { toast } from 'sonner';

export function QADetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: qa, isLoading, isError } = useQADetail(id!);
  const updateQA = useUpdateQA();
  const deleteQA = useDeleteQA();

  const [editing, setEditing] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [question, setQuestion] = useState('');
  const [answer, setAnswer] = useState('');
  const [isFaq, setIsFaq] = useState(false);

  const startEditing = () => {
    if (!qa) return;
    setQuestion(qa.question);
    setAnswer(qa.answer);
    setIsFaq(qa.is_faq);
    setEditing(true);
  };

  const handleSave = () => {
    updateQA.mutate(
      { id: id!, data: { question, answer, is_faq: isFaq } },
      {
        onSuccess: () => {
          setEditing(false);
          toast.success('Пара Q&A обновлена');
        },
        onError: () => toast.error('Не удалось обновить'),
      }
    );
  };

  const handleDelete = () => {
    deleteQA.mutate(id!, {
      onSuccess: () => {
        toast.success('Пара Q&A удалена');
        navigate('/kb/qa');
      },
      onError: () => toast.error('Не удалось удалить'),
    });
  };

  if (isLoading) return <LoadingState />;
  if (isError || !qa) return <ErrorState message="Пара Q&A не найдена." />;

  return (
    <div className="space-y-6 max-w-4xl">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate('/kb/qa')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-2xl font-semibold flex-1">Детали Q&A</h1>
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
            <Button onClick={handleSave} disabled={updateQA.isPending}>
              <Save className="h-4 w-4 mr-2" />
              Сохранить
            </Button>
            <Button variant="outline" onClick={() => setEditing(false)}>
              <X className="h-4 w-4 mr-2" />
              Отмена
            </Button>
          </div>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Вопрос</CardTitle>
        </CardHeader>
        <CardContent>
          {editing ? (
            <Input value={question} onChange={(e) => setQuestion(e.target.value)} />
          ) : (
            <p>{qa.question}</p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Ответ</CardTitle>
        </CardHeader>
        <CardContent>
          {editing ? (
            <Textarea value={answer} onChange={(e) => setAnswer(e.target.value)} rows={6} />
          ) : (
            <p className="whitespace-pre-wrap">{qa.answer}</p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-4 flex-wrap">
            <div className="flex items-center gap-2">
              <Label>FAQ</Label>
              {editing ? (
                <input
                  type="checkbox"
                  checked={isFaq}
                  onChange={(e) => setIsFaq(e.target.checked)}
                  className="h-4 w-4"
                />
              ) : (
                <Badge variant={qa.is_faq ? 'secondary' : 'outline'}>
                  {qa.is_faq ? 'Да' : 'Нет'}
                </Badge>
              )}
            </div>
            <Separator orientation="vertical" className="h-6" />
            <div className="flex items-center gap-2">
              <Label>Заблокирован</Label>
              <Badge variant={qa.is_locked ? 'destructive' : 'outline'}>
                {qa.is_locked ? 'Да' : 'Нет'}
              </Badge>
            </div>
            <Separator orientation="vertical" className="h-6" />
            <span className="text-sm text-muted-foreground">
              Обновлено: {new Date(qa.updated_at).toLocaleString()}
            </span>
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <CommentsPanel entityType="qa" entityId={id!} />
        <LinksPanel entityType="qa" entityId={id!} />
      </div>

      <ConfirmDialog
        open={showDelete}
        onOpenChange={setShowDelete}
        title="Удалить пару Q&A"
        description="Вы уверены, что хотите удалить эту пару Q&A? Это действие нельзя отменить."
        onConfirm={handleDelete}
        loading={deleteQA.isPending}
      />
    </div>
  );
}
