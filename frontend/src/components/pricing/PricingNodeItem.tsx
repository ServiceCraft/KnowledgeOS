import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ChevronRight, Pencil, Trash2, Plus } from 'lucide-react';
import type { PricingNode } from '@/types';

interface TreeNode extends PricingNode {
  children: TreeNode[];
}

interface PricingNodeItemProps {
  node: TreeNode;
  onEdit: (node: PricingNode) => void;
  onDelete: (id: string) => void;
  onAddChild: (parentId: string) => void;
  depth?: number;
}

export function PricingNodeItem({
  node,
  onEdit,
  onDelete,
  onAddChild,
  depth = 0,
}: PricingNodeItemProps) {
  const [open, setOpen] = useState(true);
  const hasChildren = node.children.length > 0;

  return (
    <div style={{ marginLeft: depth > 0 ? 20 : 0 }}>
      <Collapsible open={open} onOpenChange={setOpen}>
        <div className="flex items-center gap-2 py-1.5 px-2 rounded-md hover:bg-muted/50 group">
          {hasChildren ? (
            <CollapsibleTrigger
              render={<Button variant="ghost" size="icon" className="h-6 w-6" />}
            >
              <ChevronRight
                className={`h-4 w-4 transition-transform ${open ? 'rotate-90' : ''}`}
              />
            </CollapsibleTrigger>
          ) : (
            <div className="w-6" />
          )}

          <Badge variant="outline" className="text-xs capitalize">
            {node.node_type}
          </Badge>
          <span className="font-medium text-sm flex-1">{node.name}</span>
          {node.price != null && (
            <span className="text-sm text-muted-foreground font-mono">
              {node.price.toFixed(2)} ₽
            </span>
          )}

          <div className="opacity-0 group-hover:opacity-100 flex gap-1">
            <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => onAddChild(node.id)}>
              <Plus className="h-3 w-3" />
            </Button>
            <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => onEdit(node)}>
              <Pencil className="h-3 w-3" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 text-destructive"
              onClick={() => onDelete(node.id)}
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
        </div>

        {hasChildren && (
          <CollapsibleContent>
            {node.children.map((child) => (
              <PricingNodeItem
                key={child.id}
                node={child}
                onEdit={onEdit}
                onDelete={onDelete}
                onAddChild={onAddChild}
                depth={depth + 1}
              />
            ))}
          </CollapsibleContent>
        )}
      </Collapsible>
    </div>
  );
}
