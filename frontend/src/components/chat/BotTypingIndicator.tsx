import { Bot } from 'lucide-react';

export function BotTypingIndicator() {
  return (
    <div className="flex items-start gap-3">
      <Bot className="mt-1 h-5 w-5 shrink-0 text-primary" />
      <div className="rounded-lg bg-card px-4 py-3 ring-1 ring-border">
        <div className="flex items-center gap-1.5" aria-label="Бот печатает">
          {[0, 1, 2].map((i) => (
            <span
              key={i}
              className="h-2 w-2 animate-bounce rounded-full bg-muted-foreground/60"
              style={{ animationDelay: `${i * 160}ms`, animationDuration: '0.9s' }}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
