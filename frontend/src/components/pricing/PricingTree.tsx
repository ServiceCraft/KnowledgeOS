import { useMemo } from 'react';
import type { PricingNode } from '@/types';
import { PricingNodeItem } from './PricingNodeItem';
import { EmptyState } from '@/components/shared/EmptyState';

interface TreeNode extends PricingNode {
  children: TreeNode[];
}

interface PricingTreeProps {
  nodes: PricingNode[];
  query?: string;
  onEdit: (node: PricingNode) => void;
  onDelete: (id: string) => void;
  onAddChild: (parentId: string) => void;
}

function filterTree(roots: TreeNode[], q: string): TreeNode[] {
  const lower = q.toLowerCase();
  function keep(node: TreeNode): TreeNode | null {
    const filteredChildren = node.children.map(keep).filter(Boolean) as TreeNode[];
    const selfMatch = node.name.toLowerCase().includes(lower);
    if (selfMatch || filteredChildren.length > 0) {
      return { ...node, children: filteredChildren };
    }
    return null;
  }
  return roots.map(keep).filter(Boolean) as TreeNode[];
}

export function PricingTree({ nodes, query, onEdit, onDelete, onAddChild }: PricingTreeProps) {
  const tree = useMemo(() => {
    const map = new Map<string, TreeNode>();
    const roots: TreeNode[] = [];

    for (const node of nodes) {
      map.set(node.id, { ...node, children: [] });
    }

    for (const node of nodes) {
      const treeNode = map.get(node.id)!;
      if (node.parent_id && map.has(node.parent_id)) {
        map.get(node.parent_id)!.children.push(treeNode);
      } else {
        roots.push(treeNode);
      }
    }

    return roots;
  }, [nodes]);

  const filtered = useMemo(
    () => (query ? filterTree(tree, query) : tree),
    [tree, query],
  );

  if (tree.length === 0) {
    return <EmptyState title="Нет узлов прайсинга" message="Добавьте первую категорию или услугу." />;
  }

  if (filtered.length === 0) {
    return <EmptyState title="Ничего не найдено" message="Попробуйте изменить поисковый запрос." />;
  }

  return (
    <div className="space-y-1">
      {filtered.map((node) => (
        <PricingNodeItem
          key={node.id}
          node={node}
          onEdit={onEdit}
          onDelete={onDelete}
          onAddChild={onAddChild}
        />
      ))}
    </div>
  );
}
