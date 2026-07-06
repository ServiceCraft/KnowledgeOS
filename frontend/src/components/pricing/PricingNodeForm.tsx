import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import type { PricingNode } from '@/types';

interface PricingNodeFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: Partial<PricingNode>) => void;
  loading?: boolean;
  initialData?: PricingNode | null;
  parentId?: string;
  nodes: PricingNode[];
}

const NODE_TYPES = ['category', 'service', 'option'];
const NODE_TYPE_LABELS: Record<string, string> = {
  category: 'Категория',
  service: 'Услуга',
  option: 'Опция',
};

export function PricingNodeForm({
  open,
  onOpenChange,
  onSubmit,
  loading = false,
  initialData,
  parentId,
  nodes,
}: PricingNodeFormProps) {
  const sourceKey = `${open ? 'open' : 'closed'}:${initialData?.id ?? 'new'}:${parentId ?? ''}`;
  const sourceForm = {
    key: sourceKey,
    name: initialData?.name ?? '',
    nodeType: initialData?.node_type ?? 'category',
    price: initialData?.price != null ? String(initialData.price) : '',
    selectedParentId: initialData?.parent_id ?? parentId ?? '',
  };
  const [draft, setDraft] = useState(sourceForm);
  const form = draft.key === sourceKey ? draft : sourceForm;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      name: form.name,
      node_type: form.nodeType,
      price: form.price ? parseFloat(form.price) : undefined,
      parent_id: form.selectedParentId || undefined,
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{initialData ? 'Редактировать узел' : 'Добавить узел'}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Название</Label>
            <Input
              id="name"
              value={form.name}
              onChange={(e) => setDraft({ ...form, name: e.target.value })}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="nodeType">Тип</Label>
            <Select value={form.nodeType} onValueChange={(v) => v && setDraft({ ...form, nodeType: v })}>
              <SelectTrigger>
                <SelectValue>{NODE_TYPE_LABELS[form.nodeType] ?? form.nodeType}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {NODE_TYPES.map((t) => (
                  <SelectItem key={t} value={t}>
                    {NODE_TYPE_LABELS[t] ?? t}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="price">Цена</Label>
            <Input
              id="price"
              type="number"
              step="0.01"
              value={form.price}
              onChange={(e) => setDraft({ ...form, price: e.target.value })}
              placeholder="Необязательно"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="parent">Родительский узел</Label>
            <Select value={form.selectedParentId} onValueChange={(v) => setDraft({ ...form, selectedParentId: v ?? '' })}>
              <SelectTrigger>
                <SelectValue placeholder="Нет (корневой)">
                  {form.selectedParentId ? nodes.find((n) => n.id === form.selectedParentId)?.name ?? 'Нет (корневой)' : 'Нет (корневой)'}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">Нет (корневой)</SelectItem>
                {nodes
                  .filter((n) => n.id !== initialData?.id)
                  .map((n) => (
                    <SelectItem key={n.id} value={n.id}>
                      {n.name}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" disabled={loading || !form.name.trim()}>
              {loading ? 'Сохранение...' : 'Сохранить'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
