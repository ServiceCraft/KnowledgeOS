import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { NumberInput } from '@/components/ui/number-input';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { toast } from 'sonner';
import type { BotSettings } from '@/types';
import { useBotSettings, useUpdateBotSettings } from '@/hooks/useBotAdmin';
import {
  BOOL_LABELS,
  MODEL_TIER_LABELS,
  SETTING_HINTS,
} from '@/components/bot-admin/constants';
import { FieldHint } from '@/components/bot-admin/FieldHint';
import { handoffModule, handoffFallbackText, yclientsBookingModule } from '@/components/bot-admin/moduleHelpers';

export function SettingsTab() {
  const { data, isLoading, isError } = useBotSettings();
  const update = useUpdateBotSettings();
  const [draft, setDraft] = useState<Partial<BotSettings>>({});

  if (isLoading) return <LoadingState />;
  if (isError || !data) return <ErrorState message="Не удалось загрузить настройки бота." />;

  const form: BotSettings = { ...data, ...draft };

  const save = () => {
    update.mutate(
      {
        enabled: form.enabled,
        provider: form.provider,
        model_tier: form.model_tier,
        model: form.model || undefined,
        temperature: form.temperature,
        max_tokens: form.max_tokens,
        persona_name: form.persona_name,
        persona_tone: form.persona_tone,
        persona_rules: form.persona_rules,
        enabled_modules: form.enabled_modules,
        min_retrieval_score: form.min_retrieval_score,
        min_confidence: form.min_confidence,
        escalate_on_low_confidence: form.escalate_on_low_confidence,
        require_citations: form.require_citations,
        allowed_theme_ids: Array.isArray(form.allowed_theme_ids) ? form.allowed_theme_ids : [],
      },
      {
        onSuccess: () => {
          setDraft({});
          toast.success('Настройки сохранены');
        },
        onError: () => toast.error('Не удалось сохранить настройки'),
      }
    );
  };

  const setHandoffEnabled = (enabled: boolean) => {
    setDraft((f) => ({
      ...f,
      enabled_modules: {
        ...(f.enabled_modules ?? data.enabled_modules ?? {}),
        handoff: enabled,
      },
    }));
  };

  const setHandoffText = (text: string) => {
    setDraft((f) => ({
      ...f,
      enabled_modules: {
        ...(f.enabled_modules ?? data.enabled_modules ?? {}),
        handoff_fallback_text: text,
      },
    }));
  };

  const setYClientsBooking = (enabled: boolean) => {
    setDraft((f) => ({
      ...f,
      enabled_modules: {
        ...(f.enabled_modules ?? data.enabled_modules ?? {}),
        yclients_booking: enabled,
      },
    }));
  };

  return (
    <div className="space-y-6 max-w-2xl">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Основные</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3">
            <FieldHint label="Бот включён" hint={SETTING_HINTS.enabled} />
            <Select
              value={form.enabled ? 'true' : 'false'}
              onValueChange={(v) => setDraft((f) => ({ ...f, enabled: v === 'true' }))}
            >
              <SelectTrigger className="w-32">
                <SelectValue>
                  {form.enabled ? BOOL_LABELS.true : BOOL_LABELS.false}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="true">{BOOL_LABELS.true}</SelectItem>
                <SelectItem value="false">{BOOL_LABELS.false}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <FieldHint label="Тариф модели" hint={SETTING_HINTS.model_tier} />
              <Select
                value={form.model_tier ?? 'lite'}
                onValueChange={(v) => setDraft((f) => ({ ...f, model_tier: v as BotSettings['model_tier'] }))}
              >
                <SelectTrigger>
                  <SelectValue>
                    {MODEL_TIER_LABELS[form.model_tier ?? 'lite']}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="lite">{MODEL_TIER_LABELS.lite}</SelectItem>
                  <SelectItem value="pro">{MODEL_TIER_LABELS.pro}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <FieldHint label="Модель" htmlFor="model" hint={SETTING_HINTS.model} />
              <Input
                id="model"
                value={form.model ?? ''}
                onChange={(e) => setDraft((f) => ({ ...f, model: e.target.value }))}
                placeholder="По умолчанию"
              />
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <FieldHint label="Температура" htmlFor="temperature" hint={SETTING_HINTS.temperature} />
              <NumberInput
                id="temperature"
                step={0.1}
                min={0}
                max={2}
                value={form.temperature ?? 0.2}
                onChange={(temperature) => setDraft((f) => ({ ...f, temperature }))}
              />
            </div>
            <div className="space-y-2">
              <FieldHint label="Макс. токенов" htmlFor="max_tokens" hint={SETTING_HINTS.max_tokens} />
              <NumberInput
                id="max_tokens"
                min={1}
                max={8192}
                value={form.max_tokens ?? 1024}
                onChange={(max_tokens) => setDraft((f) => ({ ...f, max_tokens }))}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Handoff оператору</CardTitle>
          <CardDescription>Очередь эскалаций и ручной перехват диалогов оператором</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3">
            <FieldHint label="Модуль handoff включён" htmlFor="handoff-enabled" hint={SETTING_HINTS.handoff_enabled} />
            <input
              id="handoff-enabled"
              type="checkbox"
              checked={handoffModule(form.enabled_modules)}
              onChange={(e) => setHandoffEnabled(e.target.checked)}
            />
          </div>
          <div className="space-y-2">
            <FieldHint label="Текст при передаче оператору" htmlFor="handoff-text" hint={SETTING_HINTS.handoff_text} />
            <Textarea
              id="handoff-text"
              value={handoffFallbackText(form.enabled_modules)}
              onChange={(e) => setHandoffText(e.target.value)}
              placeholder="Передаю ваш вопрос специалисту — он свяжется с вами по этому обращению."
            />
            <p className="text-xs text-muted-foreground">
              Handoff включается только при сохранении настроек. Уведомления в Telegram-группу задаются в параметрах Telegram-канала.
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Интеграции</CardTitle>
          <CardDescription>Запись клиентов во внешние сервисы прямо из диалога</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-3">
            <FieldHint label="Запись в YClients" htmlFor="yclients-enabled" hint={SETTING_HINTS.yclients_booking} />
            <input
              id="yclients-enabled"
              type="checkbox"
              checked={yclientsBookingModule(form.enabled_modules)}
              onChange={(e) => setYClientsBooking(e.target.checked)}
            />
          </div>
          <p className="text-xs text-muted-foreground">
            Токен YClients и ID филиала задаются в разделе «Каналы и секреты» → «Сервисные секреты». Модуль включается при сохранении настроек.
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Персона</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <FieldHint label="Имя" htmlFor="persona_name" hint={SETTING_HINTS.persona_name} />
            <Input
              id="persona_name"
              value={form.persona_name ?? ''}
              onChange={(e) => setDraft((f) => ({ ...f, persona_name: e.target.value }))}
            />
          </div>
          <div className="space-y-2">
            <FieldHint label="Тон" htmlFor="persona_tone" hint={SETTING_HINTS.persona_tone} />
            <Input
              id="persona_tone"
              value={form.persona_tone ?? ''}
              onChange={(e) => setDraft((f) => ({ ...f, persona_tone: e.target.value }))}
            />
          </div>
          <div className="space-y-2">
            <FieldHint label="Правила" htmlFor="persona_rules" hint={SETTING_HINTS.persona_rules} />
            <Textarea
              id="persona_rules"
              value={form.persona_rules ?? ''}
              onChange={(e) => setDraft((f) => ({ ...f, persona_rules: e.target.value }))}
              rows={4}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Ограничения</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <FieldHint label="Мин. оценка поиска" htmlFor="min_retrieval" hint={SETTING_HINTS.min_retrieval} />
              <NumberInput
                id="min_retrieval"
                step={0.01}
                min={0}
                max={1}
                value={form.min_retrieval_score ?? 0}
                onChange={(min_retrieval_score) => setDraft((f) => ({ ...f, min_retrieval_score }))}
              />
            </div>
            <div className="space-y-2">
              <FieldHint label="Мин. уверенность" htmlFor="min_confidence" hint={SETTING_HINTS.min_confidence} />
              <NumberInput
                id="min_confidence"
                step={0.01}
                min={0}
                max={1}
                value={form.min_confidence ?? 0}
                onChange={(min_confidence) => setDraft((f) => ({ ...f, min_confidence }))}
              />
            </div>
          </div>
          <div className="flex flex-wrap gap-4">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.escalate_on_low_confidence ?? false}
                onChange={(e) => setDraft((f) => ({ ...f, escalate_on_low_confidence: e.target.checked }))}
              />
              <FieldHint label="Эскалация при низкой уверенности" hint={SETTING_HINTS.escalate} />
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.require_citations ?? false}
                onChange={(e) => setDraft((f) => ({ ...f, require_citations: e.target.checked }))}
              />
              <FieldHint label="Требовать цитирование" hint={SETTING_HINTS.citations} />
            </label>
          </div>
        </CardContent>
      </Card>

      <Button onClick={save} disabled={update.isPending}>
        {update.isPending ? 'Сохранение...' : 'Сохранить настройки'}
      </Button>
    </div>
  );
}
