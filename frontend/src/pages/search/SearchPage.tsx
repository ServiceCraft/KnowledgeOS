import { useState, useRef, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { Badge } from '@/components/ui/badge';
import {
  Command,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
} from '@/components/ui/command';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useSearch } from '@/hooks/useSearch';
import { MessageSquareQuote, FileText, Search, Loader2 } from 'lucide-react';

const TYPE_META: Record<string, { label: string; icon: typeof FileText; color: string }> = {
  qa: { label: 'Q&A', icon: MessageSquareQuote, color: 'text-blue-600' },
  article: { label: 'Статья', icon: FileText, color: 'text-amber-600' },
};

export function SearchPage() {
  const navigate = useNavigate();
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState('all');
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const types = typeFilter === 'all' ? undefined : [typeFilter];
  const { data, isFetching } = useSearch({ query: debouncedQuery, types });

  const results = data?.data ?? [];
  const total = data?.total ?? 0;
  const hasQuery = debouncedQuery.length >= 2;

  const handleValueChange = useCallback((value: string) => {
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      const trimmed = value.trim();
      setDebouncedQuery(trimmed.length >= 2 ? trimmed : '');
    }, 400);
  }, []);

  useEffect(() => {
    return () => clearTimeout(timerRef.current);
  }, []);

  const handleSelect = (value: string) => {
    // value format: "entity_type:entity_id"
    const [entityType, entityId] = value.split(':');
    if (entityType === 'qa') navigate(`/kb/qa/${entityId}`);
    else if (entityType === 'article') navigate(`/kb/articles/${entityId}`);
  };

  return (
    <div className="space-y-4 max-w-3xl">
      <div>
        <h1 className="text-2xl font-semibold">Поиск</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Поиск по вопросам и статьям
        </p>
      </div>

      <Tabs value={typeFilter} onValueChange={setTypeFilter}>
        <TabsList>
          <TabsTrigger value="all">Все</TabsTrigger>
          <TabsTrigger value="qa">Q&A</TabsTrigger>
          <TabsTrigger value="article">Статьи</TabsTrigger>
        </TabsList>
      </Tabs>

      <Command
        className="rounded-lg border bg-card shadow-sm"
        shouldFilter={false}
      >
        <CommandInput
          placeholder="Начните вводить для поиска..."
          onValueChange={handleValueChange}
          autoFocus
        />
        <CommandList className="max-h-[60vh]">
          {!hasQuery && (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Search className="h-8 w-8 text-muted-foreground/40 mb-3" />
              <p className="text-sm text-muted-foreground">Введите минимум 2 символа для поиска</p>
            </div>
          )}

          {hasQuery && !isFetching && results.length === 0 && (
            <CommandEmpty>
              Ничего не найдено по запросу "{debouncedQuery}"
            </CommandEmpty>
          )}

          {hasQuery && results.length > 0 && (
            <CommandGroup
              heading={
                <span className="flex items-center gap-2">
                  {total} {total % 10 === 1 && total % 100 !== 11 ? 'результат' : (total % 10 >= 2 && total % 10 <= 4 && (total % 100 < 10 || total % 100 >= 20)) ? 'результата' : 'результатов'}
                  {isFetching && <Loader2 className="h-3 w-3 animate-spin" />}
                </span>
              }
            >
              {results.map((result) => {
                const meta = TYPE_META[result.entity_type] ?? TYPE_META.qa;
                const Icon = meta.icon;
                return (
                  <CommandItem
                    key={`${result.entity_type}-${result.entity_id}`}
                    value={`${result.entity_type}:${result.entity_id}`}
                    onSelect={handleSelect}
                    className="flex items-start gap-3 py-3 px-3 cursor-pointer"
                  >
                    <Icon className={`h-4 w-4 mt-0.5 shrink-0 ${meta.color}`} />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm truncate">{result.title}</span>
                        <Badge variant="outline" className="text-[10px] px-1.5 py-0 shrink-0">
                          {meta.label}
                        </Badge>
                      </div>
                      <p className="text-xs text-muted-foreground line-clamp-1 mt-0.5">
                        {result.snippet}
                      </p>
                    </div>
                  </CommandItem>
                );
              })}
            </CommandGroup>
          )}

          {hasQuery && isFetching && results.length === 0 && (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          )}
        </CommandList>
      </Command>
    </div>
  );
}
