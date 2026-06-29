import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { RefreshCw } from 'lucide-react';

interface PasswordFieldWithGenerateProps {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  minLength?: number;
  hint?: string;
}

export function PasswordFieldWithGenerate({
  id,
  label,
  value,
  onChange,
  minLength = 8,
  hint,
}: PasswordFieldWithGenerateProps) {
  const generatePassword = (length = 16) => {
    const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789!@#$%';
    const arr = new Uint32Array(length);
    crypto.getRandomValues(arr);
    onChange(Array.from(arr, (n) => chars[n % chars.length]).join(''));
  };

  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <div className="flex gap-2">
        <Input
          id={id}
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          minLength={minLength}
          required
        />
        <Button type="button" variant="outline" size="icon" onClick={() => generatePassword()} title="Сгенерировать пароль">
          <RefreshCw className="h-4 w-4" />
        </Button>
      </div>
      {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
    </div>
  );
}
