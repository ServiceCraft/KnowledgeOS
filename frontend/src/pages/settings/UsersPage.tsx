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
import { useUsersList, useCreateUser, useUpdateUser, useDeleteUser } from '@/hooks/useUsers';
import type { User, Role } from '@/types';
import { toast } from 'sonner';

const ROLES: Role[] = ['viewer', 'editor', 'admin', 'superadmin'];
const ROLE_LABELS: Record<Role, string> = {
  viewer: 'Просмотр',
  editor: 'Редактор',
  admin: 'Администратор',
  superadmin: 'Суперадмин',
};

export function UsersPage() {
  const [page, setPage] = useState(1);
  const { data, isLoading, isError } = useUsersList({ page, limit: 20 });
  const createUser = useCreateUser();
  const updateUser = useUpdateUser();
  const deleteUser = useDeleteUser();

  const [showForm, setShowForm] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [formEmail, setFormEmail] = useState('');
  const [formPassword, setFormPassword] = useState('');
  const [formRole, setFormRole] = useState<Role>('viewer');
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const openCreate = () => {
    setEditingUser(null);
    setFormEmail('');
    setFormPassword('');
    setFormRole('viewer');
    setShowForm(true);
  };

  const openEdit = (user: User) => {
    setEditingUser(user);
    setFormEmail(user.email);
    setFormPassword('');
    setFormRole(user.role);
    setShowForm(true);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (editingUser) {
      updateUser.mutate(
        {
          id: editingUser.id,
          data: {
            email: formEmail,
            role: formRole,
            ...(formPassword ? { password: formPassword } : {}),
          },
        },
        {
          onSuccess: () => { setShowForm(false); toast.success('Пользователь обновлён'); },
          onError: () => toast.error('Не удалось обновить пользователя'),
        }
      );
    } else {
      createUser.mutate(
        { email: formEmail, password: formPassword, role: formRole },
        {
          onSuccess: () => { setShowForm(false); toast.success('Пользователь создан'); },
          onError: () => toast.error('Не удалось создать пользователя'),
        }
      );
    }
  };

  const handleDelete = () => {
    if (!deleteId) return;
    deleteUser.mutate(deleteId, {
      onSuccess: () => { setDeleteId(null); toast.success('Пользователь удалён'); },
      onError: () => toast.error('Не удалось удалить пользователя'),
    });
  };

  const columns: Column<User>[] = [
    {
      key: 'email',
      header: 'Email',
      render: (item) => <span className="font-medium">{item.email}</span>,
    },
    {
      key: 'role',
      header: 'Роль',
      className: 'w-32',
      render: (item) => (
        <Badge variant="secondary">
          {ROLE_LABELS[item.role] ?? item.role}
        </Badge>
      ),
    },
    {
      key: 'created_at',
      header: 'Создан',
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
  if (isError) return <ErrorState message="Не удалось загрузить пользователей." />;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Пользователи</h1>
        <Button onClick={openCreate}>
          <Plus className="h-4 w-4 mr-2" />
          Добавить пользователя
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

      <Dialog open={showForm} onOpenChange={setShowForm}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingUser ? 'Редактировать пользователя' : 'Создать пользователя'}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                value={formEmail}
                onChange={(e) => setFormEmail(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">
                Пароль {editingUser && '(оставьте пустым, чтобы сохранить текущий)'}
              </Label>
              <Input
                id="password"
                type="password"
                value={formPassword}
                onChange={(e) => setFormPassword(e.target.value)}
                required={!editingUser}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="role">Роль</Label>
              <Select value={formRole} onValueChange={(v) => v && setFormRole(v as Role)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ROLES.map((r) => (
                    <SelectItem key={r} value={r}>
                      {ROLE_LABELS[r]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setShowForm(false)}>
                Отмена
              </Button>
              <Button type="submit" disabled={createUser.isPending || updateUser.isPending}>
                {editingUser ? 'Сохранить' : 'Создать'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Удалить пользователя"
        description="Вы уверены, что хотите удалить этого пользователя?"
        onConfirm={handleDelete}
        loading={deleteUser.isPending}
      />
    </div>
  );
}
