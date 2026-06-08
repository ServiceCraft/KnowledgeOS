import { useEffect, useRef } from 'react';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { useCallDetail } from '@/hooks/useCalls';

interface CallTranscriptSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  callId: string | null;
  highlightStart?: number;
  highlightEnd?: number;
}

export function CallTranscriptSheet({
  open,
  onOpenChange,
  callId,
  highlightStart,
  highlightEnd,
}: CallTranscriptSheetProps) {
  const { data: call, isLoading, isError } = useCallDetail(open ? callId : null);
  const highlightRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!call || highlightStart === undefined || highlightEnd === undefined) return;
    const id = window.setTimeout(() => {
      highlightRef.current?.scrollIntoView({ block: 'center', behavior: 'smooth' });
    }, 50);
    return () => window.clearTimeout(id);
  }, [call, highlightStart, highlightEnd]);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="!w-full sm:!max-w-[50vw] flex flex-col"
      >
        <SheetHeader className="border-b">
          <SheetTitle>{call?.title || 'Транскрипт звонка'}</SheetTitle>
          {call?.occurred_at && (
            <SheetDescription>
              {new Date(call.occurred_at).toLocaleString('ru-RU')}
              {call.duration_sec ? ` · ${formatDuration(call.duration_sec)}` : ''}
            </SheetDescription>
          )}
        </SheetHeader>

        <div className="flex-1 overflow-y-auto p-4 text-sm leading-relaxed whitespace-pre-wrap">
          {isLoading && (
            <p className="text-muted-foreground">Загрузка транскрипта…</p>
          )}
          {isError && (
            <p className="text-destructive">Не удалось загрузить транскрипт.</p>
          )}
          {call && (
            <TranscriptBody
              transcript={call.transcript}
              highlightStart={highlightStart}
              highlightEnd={highlightEnd}
              highlightRef={highlightRef}
            />
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

interface TranscriptBodyProps {
  transcript: string;
  highlightStart?: number;
  highlightEnd?: number;
  highlightRef: React.RefObject<HTMLElement | null>;
}

function TranscriptBody({
  transcript,
  highlightStart,
  highlightEnd,
  highlightRef,
}: TranscriptBodyProps) {
  const hasHighlight =
    highlightStart !== undefined &&
    highlightEnd !== undefined &&
    highlightStart >= 0 &&
    highlightEnd > highlightStart &&
    highlightStart < transcript.length;

  if (!transcript) {
    return <p className="text-muted-foreground italic">Транскрипт пуст.</p>;
  }

  if (!hasHighlight) {
    return <>{transcript}</>;
  }

  const start = highlightStart!;
  const end = Math.min(highlightEnd!, transcript.length);
  const before = transcript.slice(0, start);
  const match = transcript.slice(start, end);
  const after = transcript.slice(end);

  return (
    <>
      {before}
      <mark
        ref={highlightRef}
        className="bg-yellow-200 dark:bg-yellow-900/60 rounded px-0.5"
      >
        {match}
      </mark>
      {after}
    </>
  );
}

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}
