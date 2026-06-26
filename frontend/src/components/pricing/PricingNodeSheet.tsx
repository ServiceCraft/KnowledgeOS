import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet';
import { Badge } from '@/components/ui/badge';
import { CommentsPanel } from '@/components/shared/CommentsPanel';
import { LinksPanel } from '@/components/shared/LinksPanel';
import type { PricingNode } from '@/types';

interface PricingNodeSheetProps {
  node: PricingNode | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

const NODE_TYPE_LABELS: Record<string, string> = {
  category: 'Категория',
  service: 'Услуга',
  option: 'Опция',
};

export function PricingNodeSheet({ node, open, onOpenChange }: PricingNodeSheetProps) {
  if (!node) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="!w-full sm:!max-w-xl overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{node.name}</SheetTitle>
          <SheetDescription className="flex items-center gap-2">
            <Badge variant="outline">{NODE_TYPE_LABELS[node.node_type] ?? node.node_type}</Badge>
            {node.price != null && <span>{node.price.toFixed(2)} ₽</span>}
          </SheetDescription>
        </SheetHeader>

        <div className="mt-6 space-y-6">
          <CommentsPanel entityType="pricing" entityId={node.id} />
          <LinksPanel entityType="pricing" entityId={node.id} />
        </div>
      </SheetContent>
    </Sheet>
  );
}
