import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Plus } from 'lucide-react';
import { PricingTree } from '@/components/pricing/PricingTree';
import { PricingNodeForm } from '@/components/pricing/PricingNodeForm';
import { SearchInput } from '@/components/shared/SearchInput';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { usePricingList, useCreatePricingNode, useUpdatePricingNode, useDeletePricingNode } from '@/hooks/usePricing';
import type { PricingNode } from '@/types';
import { toast } from 'sonner';

export function PricingPage() {
  const { data, isLoading, isError } = usePricingList({ limit: 1000 });
  const createNode = useCreatePricingNode();
  const updateNode = useUpdatePricingNode();
  const deleteNode = useDeletePricingNode();

  const [query, setQuery] = useState('');
  const [showForm, setShowForm] = useState(false);
  const [editingNode, setEditingNode] = useState<PricingNode | null>(null);
  const [parentId, setParentId] = useState<string | undefined>();
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const nodes = data?.data ?? [];

  const handleAddRoot = () => {
    setEditingNode(null);
    setParentId(undefined);
    setShowForm(true);
  };

  const handleAddChild = (pid: string) => {
    setEditingNode(null);
    setParentId(pid);
    setShowForm(true);
  };

  const handleEdit = (node: PricingNode) => {
    setEditingNode(node);
    setParentId(node.parent_id ?? undefined);
    setShowForm(true);
  };

  const handleSubmit = (formData: Partial<PricingNode>) => {
    if (editingNode) {
      updateNode.mutate(
        { id: editingNode.id, data: formData },
        {
          onSuccess: () => { setShowForm(false); toast.success('Узел обновлён'); },
          onError: () => toast.error('Не удалось обновить узел'),
        }
      );
    } else {
      createNode.mutate(formData, {
        onSuccess: () => { setShowForm(false); toast.success('Узел создан'); },
        onError: () => toast.error('Не удалось создать узел'),
      });
    }
  };

  const handleDelete = () => {
    if (!deleteId) return;
    deleteNode.mutate(deleteId, {
      onSuccess: () => { setDeleteId(null); toast.success('Узел удалён'); },
      onError: () => toast.error('Не удалось удалить узел'),
    });
  };

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message="Не удалось загрузить данные прайса." />;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Прайс</h1>
        <Button onClick={handleAddRoot}>
          <Plus className="h-4 w-4 mr-2" />
          Добавить корневой узел
        </Button>
      </div>

      <SearchInput
        onSearch={setQuery}
        placeholder="Поиск по прайсу..."
        className="max-w-sm"
      />

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Дерево прайса</CardTitle>
        </CardHeader>
        <CardContent>
          <PricingTree
            nodes={nodes}
            query={query}
            onEdit={handleEdit}
            onDelete={(id) => setDeleteId(id)}
            onAddChild={handleAddChild}
          />
        </CardContent>
      </Card>

      <PricingNodeForm
        open={showForm}
        onOpenChange={setShowForm}
        onSubmit={handleSubmit}
        loading={createNode.isPending || updateNode.isPending}
        initialData={editingNode}
        parentId={parentId}
        nodes={nodes}
      />

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Удалить узел"
        description="Вы уверены, что хотите удалить этот узел прайса и все его дочерние элементы?"
        onConfirm={handleDelete}
        loading={deleteNode.isPending}
      />
    </div>
  );
}
