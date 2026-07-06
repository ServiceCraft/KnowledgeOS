import { useState } from 'react';
import { Bot, Check, Pencil, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import type { ReviewRequest } from '@/api/qa';
import type { QAPair } from '@/types';

interface AIReviewPanelProps {
  item: QAPair;
  canWrite: boolean;
  isPending: boolean;
  onAccept: () => void;
  onReject: () => void;
  onEditSave: (editedAnswer: string) => void;
}

export function AIReviewPanel({
  item,
  canWrite,
  isPending,
  onAccept,
  onReject,
  onEditSave,
}: AIReviewPanelProps) {
  const [editingAI, setEditingAI] = useState(false);
  const sourceAIAnswer = item.ai_answer || '';
  const [editedAI, setEditedAI] = useState<{ source: string; value: string } | null>(null);
  const editedAIAnswer = editedAI?.source === sourceAIAnswer ? editedAI.value : sourceAIAnswer;

  if (item.ai_status !== 'pending' || !canWrite) return null;

  return (
    <div className="rounded-md border border-violet-200 bg-violet-50 dark:border-violet-800 dark:bg-violet-950/30 p-3 space-y-2">
      <div className="flex items-center gap-2">
        <Bot className="h-4 w-4 text-violet-600 dark:text-violet-400" />
        <span className="text-xs font-medium text-violet-700 dark:text-violet-300">AI-предложенный ответ</span>
      </div>
      {item.answer && (
        <div className="text-xs text-muted-foreground bg-muted/50 rounded p-2">
          <span className="font-medium">Текущий ответ: </span>
          {item.answer}
        </div>
      )}
      {editingAI ? (
        <div className="space-y-2">
          <Textarea
            value={editedAIAnswer}
            onChange={(e) => setEditedAI({ source: sourceAIAnswer, value: e.target.value })}
            rows={3}
            className="text-sm resize-y"
          />
          <div className="flex gap-2">
            <Button
              size="sm"
              onClick={() => onEditSave(editedAIAnswer)}
              disabled={isPending}
              className="bg-green-600 hover:bg-green-700 text-white"
            >
              {isPending ? 'Сохранение...' : 'Сохранить'}
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                setEditingAI(false);
                setEditedAI(null);
              }}
            >
              Отмена
            </Button>
          </div>
        </div>
      ) : (
        <>
          <p className="text-sm whitespace-pre-wrap">{item.ai_answer}</p>
          <div className="flex gap-2 pt-1">
            <Button
              size="sm"
              variant="outline"
              className="text-red-600 border-red-200 hover:bg-red-50 dark:border-red-800 dark:hover:bg-red-950/30"
              onClick={onReject}
              disabled={isPending}
            >
              <X className="h-3.5 w-3.5 mr-1" />
              Отклонить
            </Button>
            <Button
              size="sm"
              className="bg-green-600 hover:bg-green-700 text-white"
              onClick={onAccept}
              disabled={isPending}
            >
              <Check className="h-3.5 w-3.5 mr-1" />
              Принять
            </Button>
            <Button
              size="sm"
              variant="outline"
              className="text-yellow-600 border-yellow-200 hover:bg-yellow-50 dark:border-yellow-800 dark:hover:bg-yellow-950/30"
              onClick={() => setEditingAI(true)}
              disabled={isPending}
            >
              <Pencil className="h-3.5 w-3.5 mr-1" />
              Править
            </Button>
          </div>
        </>
      )}
    </div>
  );
}

export type QARowMutations = {
  onSaveAnswer: (id: string, answer: string) => void;
  onReviewAI: (id: string, data: ReviewRequest) => void;
  isSavingAnswer: boolean;
  isReviewing: boolean;
};
