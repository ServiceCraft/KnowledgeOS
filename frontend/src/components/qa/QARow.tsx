import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Textarea } from '@/components/ui/textarea';
import { Star, ExternalLink, Check, X, Pencil } from 'lucide-react';
import { AIReviewPanel } from '@/components/qa/AIReviewPanel';
import type { QAPair } from '@/types';
import type { QARowMutations } from '@/components/qa/AIReviewPanel';

function FrequencyBadge({ value }: { value: number }) {
  let color = 'bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400';
  if (value >= 10) color = 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400';
  else if (value >= 5) color = 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-400';
  else if (value >= 1) color = 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400';

  return (
    <div className={`shrink-0 rounded-lg px-2.5 py-1.5 flex items-center justify-center font-bold text-lg tabular-nums ${color}`}>
      {value} <span className="text-xs font-normal ml-1 opacity-70">раз</span>
    </div>
  );
}

function AIStatusBadge({ status }: { status: string }) {
  switch (status) {
    case 'accepted':
      return <Badge className="text-xs gap-1 px-1.5 py-0 bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400"><Check className="h-3 w-3" />AI принят</Badge>;
    case 'rejected':
      return <Badge className="text-xs gap-1 px-1.5 py-0 bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-400"><X className="h-3 w-3" />AI отклонён</Badge>;
    case 'edited':
      return <Badge className="text-xs gap-1 px-1.5 py-0 bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400"><Pencil className="h-3 w-3" />AI исправлен</Badge>;
    default:
      return null;
  }
}

interface QARowProps extends QARowMutations {
  item: QAPair;
  themeName: string | null;
  onNavigate: () => void;
  canWrite: boolean;
}

export function QARow({
  item,
  themeName,
  onNavigate,
  canWrite,
  onSaveAnswer,
  onReviewAI,
  isSavingAnswer,
  isReviewing,
}: QARowProps) {
  const sourceAnswer = item.answer || '';
  const [answerDraft, setAnswerDraft] = useState<{ source: string; value: string } | null>(null);
  const answer = answerDraft?.source === sourceAnswer ? answerDraft.value : sourceAnswer;
  const answerDirty = answer !== sourceAnswer;
  const isPending = item.ai_status === 'pending';

  return (
    <div className="rounded-lg border bg-card p-4 space-y-2">
      <div className="flex items-center gap-3">
        <FrequencyBadge value={item.frequency} />
        <div className="flex-1 min-w-0">
          <p className="font-medium text-sm leading-snug">{item.question}</p>
          <div className="flex items-center gap-2 mt-1">
            {item.is_faq && (
              <Badge variant="secondary" className="text-xs gap-1 px-1.5 py-0">
                <Star className="h-3 w-3" />
                FAQ
              </Badge>
            )}
            {item.is_locked && (
              <Badge variant="outline" className="text-xs px-1.5 py-0">Заблокирован</Badge>
            )}
            {item.ai_status && item.ai_status !== 'pending' && (
              <AIStatusBadge status={item.ai_status} />
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
        <Button
          variant="ghost"
          size="icon"
          className="shrink-0 h-8 w-8"
          onClick={onNavigate}
          title="Открыть детали"
        >
          <ExternalLink className="h-4 w-4 text-muted-foreground" />
        </Button>
      </div>

      <AIReviewPanel
        item={item}
        canWrite={canWrite}
        isPending={isReviewing}
        onAccept={() => onReviewAI(item.id, { action: 'accept' })}
        onReject={() => onReviewAI(item.id, { action: 'reject' })}
        onEditSave={(editedAnswer) => onReviewAI(item.id, { action: 'edit', edited_answer: editedAnswer })}
      />

      {(!isPending || !canWrite) &&
        (canWrite ? (
          <div className="flex gap-2 items-start">
            <Textarea
              value={answer}
              onChange={(e) => {
                setAnswerDraft({ source: sourceAnswer, value: e.target.value });
              }}
              placeholder="Введите ответ..."
              rows={2}
              className="text-sm flex-1 min-h-[2.5rem] resize-y"
            />
            {answerDirty && (
              <Button
                size="sm"
                className="shrink-0 h-9 px-4 bg-green-600 hover:bg-green-700 text-white"
                onClick={() => onSaveAnswer(item.id, answer)}
                disabled={isSavingAnswer}
              >
                {isSavingAnswer ? 'Сохранение...' : 'Сохранить'}
              </Button>
            )}
          </div>
        ) : (
          <div className="text-sm whitespace-pre-wrap">
            {item.answer || <span className="text-muted-foreground">Ответ не задан</span>}
          </div>
        ))}
    </div>
  );
}
