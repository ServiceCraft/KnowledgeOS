import { useState, type ComponentProps, type FocusEvent } from 'react';
import { Input } from '@/components/ui/input';

type RevealableSecretInputProps = Omit<ComponentProps<typeof Input>, 'type'> & {
  /** When false, renders a plain text input without focus-based masking. */
  secret?: boolean;
};

export function RevealableSecretInput({
  secret = true,
  className,
  onFocus,
  onBlur,
  ...props
}: RevealableSecretInputProps) {
  const [focused, setFocused] = useState(false);

  if (!secret) {
    return <Input type="text" className={className} onFocus={onFocus} onBlur={onBlur} {...props} />;
  }

  const handleFocus = (event: FocusEvent<HTMLInputElement>) => {
    setFocused(true);
    onFocus?.(event);
  };

  const handleBlur = (event: FocusEvent<HTMLInputElement>) => {
    setFocused(false);
    onBlur?.(event);
  };

  return (
    <Input
      type={focused ? 'text' : 'password'}
      className={className}
      onFocus={handleFocus}
      onBlur={handleBlur}
      {...props}
    />
  );
}
