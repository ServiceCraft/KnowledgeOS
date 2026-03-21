import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
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
import { Plus, Pencil, Trash2 } from 'lucide-react';
import { DataTable, type Column } from '@/components/shared/DataTable';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import { useCompaniesList, useCreateCompany, useUpdateCompany, useDeleteCompany } from '@/hooks/useAdmin';
import type { Company } from '@/types';
import { toast } from 'sonner';

const TIERS = ['local', 'cloud', 'enterprise'];

export function CompaniesPage() {
  const [page, setPage] = useState(1);
  const { data, isLoading, isError } = useCompaniesList({ page, limit: 20 });
  const createCompany = useCreateCompany();
  const updateCompany = useUpdateCompany();
  const deleteCompany = useDeleteCompany();

  const [showCreate, setShowCreate] = useState(false);
  const [showEdit, setShowEdit] = useState(false);
  const [editingCompany, setEditingCompany] = useState<Company | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const [formName, setFormName] = useState('');
  const [formTier, setFormTier] = useState('local');
  const [formAdminEmail, setFormAdminEmail] = useState('');
  const [formAdminPassword, setFormAdminPassword] = useState('');

  const openCreate = () => {
    setFormName('');
    setFormTier('local');
    setFormAdminEmail('');
    setFormAdminPassword('');
    setShowCreate(true);
  };

  const openEdit = (company: Company) => {
    setEditingCompany(company);
    setFormName(company.name);
    setFormTier(company.tier);
    setShowEdit(true);
  };

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    createCompany.mutate(
      {
        name: formName,
        tier: formTier,
        admin_email: formAdminEmail,
        admin_password: formAdminPassword,
      },
      {
        onSuccess: () => { setShowCreate(false); toast.success('Компания создана'); },
        onError: () => toast.error('Не удалось создать компанию'),
      }
    );
  };

  const handleUpdate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingCompany) return;
    updateCompany.mutate(
      { id: editingCompany.id, data: { name: formName, tier: formTier } },
      {
        onSuccess: () => { setShowEdit(false); toast.success('Компания обновлена'); },
        onError: () => toast.error('Не удалось обновить компанию'),
      }
    );
  };

  const handleDelete = () => {
    if (!deleteId) return;
    deleteCompany.mutate(deleteId, {
      onSuccess: () => { setDeleteId(null); toast.success('Компания удалена'); },
      onError: () => toast.error('Не удалось удалить компанию'),
    });
  };

  const columns: Column<Company>[] = [
    {
      key: 'name',
      header: 'Название',
      render: (item) => <span className="font-medium">{item.name}</span>,
    },
    {
      key: 'tier',
      header: 'Тариф',
      className: 'w-32',
      render: (item) => (
        <Badge variant="secondary" className="capitalize">
          {item.tier}
        </Badge>
      ),
    },
    {
      key: 'created_at',
      header: 'Создана',
      className: 'w-40',
      render: (item) => (
        <span className="text-sm text-muted-foreground">
          {new Date(item.created_at).toLocaleDateString()}
        </span>
      ),
    },
    {
      key: 'actions',
      header: '',
      className: 'w-24',
      render: (item) => (
        <div className="flex gap-1" onClick={(e) => e.stopPropagation()}>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => openEdit(item)}>
            <Pencil className="h-3 w-3" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-destructive"
            onClick={() => setDeleteId(item.id)}
          >
            <Trash2 className="h-3 w-3" />
          </Button>
        </div>
      ),
    },
  ];

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message="Не удалось загрузить компании." />;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Компании</h1>
        <Button onClick={openCreate}>
          <Plus className="h-4 w-4 mr-2" />
          Добавить компанию
        </Button>
      </div>

      <DataTable
        columns={columns}
        data={data?.data ?? []}
        total={data?.total ?? 0}
        page={page}
        limit={20}
        onPageChange={setPage}
      />

      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Создать компанию</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4">
            <div className="space-y-2">
              <Label>Название компании</Label>
              <Input value={formName} onChange={(e) => setFormName(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label>Тариф</Label>
              <Select value={formTier} onValueChange={(v) => v && setFormTier(v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TIERS.map((t) => (
                    <SelectItem key={t} value={t} className="capitalize">
                      {t}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Email администратора</Label>
              <Input
                type="email"
                value={formAdminEmail}
                onChange={(e) => setFormAdminEmail(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label>Пароль администратора</Label>
              <Input
                type="password"
                value={formAdminPassword}
                onChange={(e) => setFormAdminPassword(e.target.value)}
                required
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>
                Отмена
              </Button>
              <Button type="submit" disabled={createCompany.isPending}>
                Создать
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={showEdit} onOpenChange={setShowEdit}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Редактировать компанию</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleUpdate} className="space-y-4">
            <div className="space-y-2">
              <Label>Название компании</Label>
              <Input value={formName} onChange={(e) => setFormName(e.target.value)} required />
            </div>
            <div className="space-y-2">
              <Label>Тариф</Label>
              <Select value={formTier} onValueChange={(v) => v && setFormTier(v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {TIERS.map((t) => (
                    <SelectItem key={t} value={t} className="capitalize">
                      {t}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setShowEdit(false)}>
                Отмена
              </Button>
              <Button type="submit" disabled={updateCompany.isPending}>
                Сохранить
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Удалить компанию"
        description="Вы уверены? Компания и все её данные будут удалены безвозвратно."
        onConfirm={handleDelete}
        loading={deleteCompany.isPending}
      />
    </div>
  );
}
