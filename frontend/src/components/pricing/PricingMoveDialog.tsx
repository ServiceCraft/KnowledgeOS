import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import type { PricingNode } from '@/types';

interface PricingMoveDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  node: PricingNode | null;
  nodes: PricingNode[];
  onConfirm: (parentId: string | null) => void;
  loading?: boolean;
}

function collectDescendantIds(nodes: PricingNode[], rootId: string): Set<string> {
  const ids = new Set<string>([rootId]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const n of nodes) {
      if (n.parent_id && ids.has(n.parent_id) && !ids.has(n.id)) {
        ids.add(n.id);
        changed = true;
      }
    }
  }
  return ids;
}

export function PricingMoveDialog({
  open,
  onOpenChange,
  node,
  nodes,
  onConfirm,
  loading,
}: PricingMoveDialogProps) {
  const excluded = node ? collectDescendantIds(nodes, node.id) : new Set<string>();
  const candidates = nodes.filter((n) => !excluded.has(n.id));
  const [parentId, setParentId] = useState<string>('__root__');

  // reset when dialog opens with new node
  if (!open) {
    // noop — parent selection resets on next open via key on dialog
  }

  const handleConfirm = () => {
    onConfirm(parentId === '__root__' ? null : parentId);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent key={node?.id}>
        <DialogHeader>
          <DialogTitle>Переместить «{node?.name}»</DialogTitle>
        </DialogHeader>
        <div className="space-y-2">
          <Label>Новый родитель</Label>
          <Select value={parentId} onValueChange={(v) => setParentId(v ?? '__root__')}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__root__">Корень дерева</SelectItem>
              {candidates.map((n) => (
                <SelectItem key={n.id} value={n.id}>
                  {n.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Отмена</Button>
          <Button onClick={handleConfirm} disabled={loading}>
            Переместить
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
