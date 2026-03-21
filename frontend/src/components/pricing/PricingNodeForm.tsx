import { useState, useEffect } from 'react';
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
  const [name, setName] = useState('');
  const [nodeType, setNodeType] = useState('category');
  const [price, setPrice] = useState('');
  const [selectedParentId, setSelectedParentId] = useState<string>('');

  useEffect(() => {
    if (initialData) {
      setName(initialData.name);
      setNodeType(initialData.node_type);
      setPrice(initialData.price != null ? String(initialData.price) : '');
      setSelectedParentId(initialData.parent_id ?? '');
    } else {
      setName('');
      setNodeType('category');
      setPrice('');
      setSelectedParentId(parentId ?? '');
    }
  }, [initialData, parentId, open]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      name,
      node_type: nodeType,
      price: price ? parseFloat(price) : undefined,
      parent_id: selectedParentId || undefined,
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
            <Input id="name" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="space-y-2">
            <Label htmlFor="nodeType">Тип</Label>
            <Select value={nodeType} onValueChange={(v) => v && setNodeType(v)}>
              <SelectTrigger>
                <SelectValue />
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
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              placeholder="Необязательно"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="parent">Родительский узел</Label>
            <Select value={selectedParentId} onValueChange={(v) => setSelectedParentId(v ?? '')}>
              <SelectTrigger>
                <SelectValue placeholder="Нет (корневой)" />
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
            <Button type="submit" disabled={loading || !name.trim()}>
              {loading ? 'Сохранение...' : 'Сохранить'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
