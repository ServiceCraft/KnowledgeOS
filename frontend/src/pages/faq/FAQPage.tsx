import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ChevronDown } from 'lucide-react';
import { SearchInput } from '@/components/shared/SearchInput';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { EmptyState } from '@/components/shared/EmptyState';
import { useQAList } from '@/hooks/useQA';

export function FAQPage() {
  const [query, setQuery] = useState('');
  const { data, isLoading, isError } = useQAList({
    is_faq: true,
    query: query || undefined,
    limit: 100,
  });

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message="Не удалось загрузить FAQ." />;

  const items = data?.data ?? [];

  return (
    <div className="space-y-4 max-w-3xl">
      <h1 className="text-2xl font-semibold">Часто задаваемые вопросы</h1>

      <SearchInput onSearch={setQuery} placeholder="Поиск по FAQ..." className="max-w-sm" />

      {items.length === 0 ? (
        <EmptyState title="Нет вопросов FAQ" message="Ещё нет вопросов, отмеченных как FAQ." />
      ) : (
        <div className="space-y-2">
          {items.map((item) => (
            <Collapsible key={item.id}>
              <Card>
                <CollapsibleTrigger
                  render={<CardHeader className="cursor-pointer hover:bg-muted/50 transition-colors" />}
                >
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-base font-medium">{item.question}</CardTitle>
                    <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0 transition-transform [[data-state=open]_&]:rotate-180" />
                  </div>
                </CollapsibleTrigger>
                <CollapsibleContent>
                  <CardContent className="pt-0">
                    <p className="text-muted-foreground whitespace-pre-wrap">{item.answer}</p>
                  </CardContent>
                </CollapsibleContent>
              </Card>
            </Collapsible>
          ))}
        </div>
      )}
    </div>
  );
}
