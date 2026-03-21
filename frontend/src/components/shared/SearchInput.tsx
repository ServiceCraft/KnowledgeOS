import { useState, useRef, useCallback } from 'react';
import { Input } from '@/components/ui/input';
import { Search, X, Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';

interface SearchInputProps {
  onSearch: (value: string) => void;
  placeholder?: string;
  className?: string;
  debounceMs?: number;
  minChars?: number;
  isLoading?: boolean;
  autoFocus?: boolean;
}

export function SearchInput({
  onSearch,
  placeholder = 'Поиск...',
  className,
  debounceMs = 400,
  minChars = 2,
  isLoading = false,
  autoFocus = false,
}: SearchInputProps) {
  const [hasValue, setHasValue] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const emit = useCallback(
    (raw: string) => {
      const v = raw.trim();
      onSearch(v.length >= minChars ? v : '');
    },
    [onSearch, minChars],
  );

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setHasValue(e.target.value.length > 0);
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => emit(e.target.value), debounceMs);
  };

  const handleClear = () => {
    if (inputRef.current) inputRef.current.value = '';
    setHasValue(false);
    clearTimeout(timerRef.current);
    onSearch('');
    inputRef.current?.focus();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') handleClear();
    if (e.key === 'Enter') {
      clearTimeout(timerRef.current);
      emit((e.target as HTMLInputElement).value);
    }
  };

  return (
    <div className={cn('relative', className)}>
      {isLoading ? (
        <Loader2 className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground animate-spin" />
      ) : (
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
      )}
      <Input
        ref={inputRef}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        className="pl-9 pr-8"
        autoFocus={autoFocus}
        autoComplete="off"
      />
      {hasValue && (
        <button
          type="button"
          onClick={handleClear}
          className="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded-sm hover:bg-muted text-muted-foreground"
        >
          <X className="h-3 w-3" />
        </button>
      )}
    </div>
  );
}
