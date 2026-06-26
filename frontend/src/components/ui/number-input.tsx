import { Minus, Plus } from 'lucide-react';

import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from '@/components/ui/input-group';

const numberInputClassName =
  '[appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none';

function clamp(value: number, min?: number, max?: number) {
  if (min !== undefined && value < min) return min;
  if (max !== undefined && value > max) return max;
  return value;
}

function parseStep(step: number | string | undefined) {
  if (step === undefined) return 1;
  return typeof step === 'string' ? parseFloat(step) : step;
}

function roundToStep(value: number, step: number) {
  const decimals = String(step).includes('.') ? String(step).split('.')[1]?.length ?? 0 : 0;
  return Number(value.toFixed(decimals));
}

type NumberInputProps = Omit<
  React.ComponentProps<'input'>,
  'type' | 'value' | 'onChange'
> & {
  value: number;
  onChange: (value: number) => void;
};

function NumberInput({
  value,
  onChange,
  step = 1,
  min,
  max,
  className,
  disabled,
  ...props
}: NumberInputProps) {
  const stepValue = parseStep(step);
  const minValue = min !== undefined ? Number(min) : undefined;
  const maxValue = max !== undefined ? Number(max) : undefined;

  const adjust = (delta: number) => {
    onChange(clamp(roundToStep(value + delta, stepValue), minValue, maxValue));
  };

  return (
    <InputGroup className={className}>
      <InputGroupInput
        type="number"
        value={value}
        step={step}
        min={min}
        max={max}
        disabled={disabled}
        className={numberInputClassName}
        onChange={(e) => {
          const parsed = parseFloat(e.target.value);
          if (!Number.isNaN(parsed)) {
            onChange(clamp(parsed, minValue, maxValue));
          }
        }}
        {...props}
      />
      <InputGroupAddon align="inline-end" className="gap-0 pr-1">
        <InputGroupButton
          type="button"
          size="icon-xs"
          disabled={disabled || (minValue !== undefined && value <= minValue)}
          aria-label="Уменьшить"
          onClick={() => adjust(-stepValue)}
        >
          <Minus />
        </InputGroupButton>
        <InputGroupButton
          type="button"
          size="icon-xs"
          disabled={disabled || (maxValue !== undefined && value >= maxValue)}
          aria-label="Увеличить"
          onClick={() => adjust(stepValue)}
        >
          <Plus />
        </InputGroupButton>
      </InputGroupAddon>
    </InputGroup>
  );
}

export { NumberInput, numberInputClassName };
