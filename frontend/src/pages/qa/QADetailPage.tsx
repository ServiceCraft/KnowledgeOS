import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';

import { ArrowLeft, Pencil, Trash2, Save, X, Bot, Check } from 'lucide-react';
import { useQADetail, useUpdateQA, useDeleteQA, useReviewAIAnswer } from '@/hooks/useQA';
import { usePermissions } from '@/hooks/usePermissions';
import { CommentsPanel } from '@/components/shared/CommentsPanel';
import { LinksPanel } from '@/components/shared/LinksPanel';
import { CallMentionsPanel } from '@/components/qa/CallMentionsPanel';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import { toast } from 'sonner';

export function QADetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { canWrite } = usePermissions();
  const { data: qa, isLoading, isError } = useQADetail(id!);
  const updateQA = useUpdateQA();
  const deleteQA = useDeleteQA();
  const reviewAI = useReviewAIAnswer();

  const [editing, setEditing] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [question, setQuestion] = useState('');
  const [answer, setAnswer] = useState('');
  const [isFaq, setIsFaq] = useState(false);
  const [frequency, setFrequency] = useState(0);
  const [editingAI, setEditingAI] = useState(false);
  const [editedAIAnswer, setEditedAIAnswer] = useState('');

  const startEditing = () => {
    if (!qa) return;
    setQuestion(qa.question);
    setAnswer(qa.answer || '');
    setIsFaq(qa.is_faq);
    setFrequency(qa.frequency);
    setEditing(true);
  };

  const handleSave = () => {
    updateQA.mutate(
      { id: id!, data: { question, answer, is_faq: isFaq, frequency } },
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
        {!editing && canWrite && (
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
            <Textarea value={answer} onChange={(e) => setAnswer(e.target.value)} rows={6} placeholder="Введите ответ..." />
          ) : qa.answer ? (
            <p className="whitespace-pre-wrap">{qa.answer}</p>
          ) : (
            <p className="text-muted-foreground italic">Нет ответа</p>
          )}
        </CardContent>
      </Card>

      {/* AI review section */}
      {qa.ai_status === 'pending' && qa.ai_answer && canWrite && (
        <Card className="border-violet-200 dark:border-violet-800">
          <CardHeader className="pb-2">
            <CardTitle className="text-base flex items-center gap-2">
              <Bot className="h-4 w-4 text-violet-600 dark:text-violet-400" />
              AI-предложенный ответ
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {editingAI ? (
              <div className="space-y-2">
                <Textarea
                  value={editedAIAnswer}
                  onChange={(e) => setEditedAIAnswer(e.target.value)}
                  rows={6}
                  className="resize-y"
                />
                <div className="flex gap-2">
                  <Button size="sm" onClick={() => {
                    reviewAI.mutate(
                      { id: id!, data: { action: 'edit', edited_answer: editedAIAnswer } },
                      {
                        onSuccess: () => { setEditingAI(false); toast.success('AI-ответ отредактирован и принят'); },
                        onError: () => toast.error('Не удалось сохранить'),
                      }
                    );
                  }} disabled={reviewAI.isPending} className="bg-green-600 hover:bg-green-700 text-white">
                    {reviewAI.isPending ? 'Сохранение...' : 'Сохранить'}
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => { setEditingAI(false); setEditedAIAnswer(qa.ai_answer || ''); }}>
                    Отмена
                  </Button>
                </div>
              </div>
            ) : (
              <>
                <p className="whitespace-pre-wrap text-sm bg-violet-50 dark:bg-violet-950/30 rounded-md p-3">{qa.ai_answer}</p>
                <div className="flex gap-2">
                  <Button size="sm" variant="outline" className="text-red-600 border-red-200 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950/30" onClick={() => {
                    reviewAI.mutate(
                      { id: id!, data: { action: 'reject' } },
                      {
                        onSuccess: () => toast.success('AI-ответ отклонён'),
                        onError: () => toast.error('Не удалось отклонить'),
                      }
                    );
                  }} disabled={reviewAI.isPending}>
                    <X className="h-4 w-4 mr-1" />
                    Отклонить
                  </Button>
                  <Button size="sm" className="bg-green-600 hover:bg-green-700 text-white" onClick={() => {
                    reviewAI.mutate(
                      { id: id!, data: { action: 'accept' } },
                      {
                        onSuccess: () => toast.success('AI-ответ принят'),
                        onError: () => toast.error('Не удалось принять'),
                      }
                    );
                  }} disabled={reviewAI.isPending}>
                    <Check className="h-4 w-4 mr-1" />
                    Принять
                  </Button>
                  <Button size="sm" variant="outline" className="text-yellow-600 border-yellow-200 hover:bg-yellow-50 dark:border-yellow-800 dark:hover:bg-yellow-950/30" onClick={() => { setEditedAIAnswer(qa.ai_answer || ''); setEditingAI(true); }} disabled={reviewAI.isPending}>
                    <Pencil className="h-4 w-4 mr-1" />
                    Править
                  </Button>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      )}

      {/* AI audit trail */}
      {qa.ai_status && qa.ai_status !== 'pending' && qa.ai_answer && (
        <Card className="border-muted">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm flex items-center gap-2 text-muted-foreground">
              <Bot className="h-3.5 w-3.5" />
              AI-ответ ({qa.ai_status === 'accepted' ? 'принят' : qa.ai_status === 'rejected' ? 'отклонён' : 'исправлен'})
              {qa.ai_reviewed_at && (
                <span className="ml-auto text-xs font-normal">
                  {new Date(qa.ai_reviewed_at).toLocaleString()}
                </span>
              )}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="whitespace-pre-wrap text-xs text-muted-foreground bg-muted/50 rounded-md p-2">{qa.ai_answer}</p>
          </CardContent>
        </Card>
      )}

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
              <Label>Частота</Label>
              {editing ? (
                <Input
                  type="number"
                  min={0}
                  value={frequency}
                  onChange={(e) => setFrequency(Math.max(0, parseInt(e.target.value) || 0))}
                  className="w-20 h-8"
                />
              ) : (
                <Badge variant="outline">{qa.frequency}</Badge>
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

      <CallMentionsPanel qaId={id!} />

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
