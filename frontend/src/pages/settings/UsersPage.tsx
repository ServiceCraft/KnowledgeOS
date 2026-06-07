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
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Plus, Pencil, Trash2, KeyRound, ArrowUpDown, RefreshCw } from 'lucide-react';
import { DataTable, type Column } from '@/components/shared/DataTable';
import { SearchInput } from '@/components/shared/SearchInput';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { ConfirmDialog } from '@/components/shared/ConfirmDialog';
import { useUsersList, useCreateUser, useUpdateUser, useDeleteUser } from '@/hooks/useUsers';
import { usePermissions } from '@/hooks/usePermissions';
import { ROLE_LABELS, ALL_ROLES, roleLabel } from '@/lib/roles';
import type { User, Role } from '@/types';
import { toast } from 'sonner';
import { AxiosError } from 'axios';

const PAGE_SIZE = 20;
const MIN_PASSWORD = 8;

function apiError(err: unknown, fallback: string): string {
  const ax = err as AxiosError<{ error?: string }>;
  return ax?.response?.data?.error ?? fallback;
}

function generatePassword(length = 16): string {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789!@#$%';
  const arr = new Uint32Array(length);
  crypto.getRandomValues(arr);
  return Array.from(arr, (n) => chars[n % chars.length]).join('');
}

export function UsersPage() {
  const { isSuperadmin, userId } = usePermissions();

  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState<Role | 'all'>('all');
  const [sortDesc, setSortDesc] = useState(true);

  const { data, isLoading, isError } = useUsersList({
    page,
    limit: PAGE_SIZE,
    q: search || undefined,
    role: roleFilter === 'all' ? undefined : roleFilter,
    sort: sortDesc ? '-created_at' : 'created_at',
  });

  const createUser = useCreateUser();
  const updateUser = useUpdateUser();
  const deleteUser = useDeleteUser();

  // Create / edit form
  const [showForm, setShowForm] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [formEmail, setFormEmail] = useState('');
  const [formPassword, setFormPassword] = useState('');
  const [formRole, setFormRole] = useState<Role>('viewer');
  const [formActive, setFormActive] = useState(true);
  const [formError, setFormError] = useState<string | null>(null);

  // Dangerous-action confirmations
  const [pendingSubmit, setPendingSubmit] = useState(false); // superadmin role confirm
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null);

  // Reset password flow
  const [resetTarget, setResetTarget] = useState<User | null>(null);
  const [resetPassword, setResetPassword] = useState('');
  const [resetError, setResetError] = useState<string | null>(null);

  const roleOptions: Role[] = isSuperadmin ? ALL_ROLES : ALL_ROLES.filter((r) => r !== 'superadmin');

  const openCreate = () => {
    setEditingUser(null);
    setFormEmail('');
    setFormPassword('');
    setFormRole('viewer');
    setFormActive(true);
    setFormError(null);
    setShowForm(true);
  };

  const openEdit = (user: User) => {
    setEditingUser(user);
    setFormEmail(user.email);
    setFormPassword('');
    setFormRole(user.role);
    setFormActive(user.is_active);
    setFormError(null);
    setShowForm(true);
  };

  const validateForm = (): string | null => {
    if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(formEmail.trim())) {
      return 'Введите корректный e-mail';
    }
    if (!editingUser && formPassword.length < MIN_PASSWORD) {
      return `Пароль должен содержать не менее ${MIN_PASSWORD} символов`;
    }
    return null;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const err = validateForm();
    if (err) {
      setFormError(err);
      return;
    }
    setFormError(null);

    // Confirm assigning the superadmin role (a potentially dangerous action).
    const isSuperadminAssignment =
      formRole === 'superadmin' && (!editingUser || editingUser.role !== 'superadmin');
    if (isSuperadminAssignment) {
      setPendingSubmit(true);
      return;
    }
    doSubmit();
  };

  const doSubmit = () => {
    setPendingSubmit(false);
    const isSelf = editingUser?.id === userId;

    if (editingUser) {
      updateUser.mutate(
        {
          id: editingUser.id,
          data: {
            email: formEmail.trim(),
            // Don't send role changes for your own account (server forbids it).
            ...(isSelf ? {} : { role: formRole, is_active: formActive }),
          },
        },
        {
          onSuccess: () => {
            setShowForm(false);
            toast.success('Пользователь обновлён');
          },
          onError: (er) => setFormError(apiError(er, 'Не удалось обновить пользователя')),
        }
      );
    } else {
      createUser.mutate(
        { email: formEmail.trim(), password: formPassword, role: formRole, is_active: formActive },
        {
          onSuccess: () => {
            setShowForm(false);
            toast.success('Пользователь создан');
          },
          onError: (er) => setFormError(apiError(er, 'Не удалось создать пользователя')),
        }
      );
    }
  };

  const handleDelete = () => {
    if (!deleteTarget) return;
    deleteUser.mutate(deleteTarget.id, {
      onSuccess: () => {
        setDeleteTarget(null);
        toast.success('Пользователь удалён');
      },
      onError: (er) => {
        toast.error(apiError(er, 'Не удалось удалить пользователя'));
        setDeleteTarget(null);
      },
    });
  };

  const handleResetPassword = () => {
    if (!resetTarget) return;
    if (resetPassword.length < MIN_PASSWORD) {
      setResetError(`Пароль должен содержать не менее ${MIN_PASSWORD} символов`);
      return;
    }
    updateUser.mutate(
      { id: resetTarget.id, data: { password: resetPassword } },
      {
        onSuccess: () => {
          setResetTarget(null);
          setResetPassword('');
          setResetError(null);
          toast.success('Пароль сброшен');
        },
        onError: (er) => setResetError(apiError(er, 'Не удалось сбросить пароль')),
      }
    );
  };

  const columns: Column<User>[] = [
    {
      key: 'email',
      header: 'E-mail',
      render: (item) => (
        <span className="font-medium">
          {item.email}
          {item.id === userId && <span className="ml-2 text-xs text-muted-foreground">(вы)</span>}
        </span>
      ),
    },
    {
      key: 'role',
      header: 'Роль',
      className: 'w-44',
      render: (item) => <Badge variant="secondary">{roleLabel(item.role)}</Badge>,
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
      key: 'is_active',
      header: 'Статус',
      className: 'w-28',
      render: (item) =>
        item.is_active ? (
          <Badge variant="secondary">Активен</Badge>
        ) : (
          <Badge variant="outline" className="text-muted-foreground">
            Неактивен
          </Badge>
        ),
    },
    {
      key: 'actions',
      header: '',
      className: 'w-32',
      render: (item) => {
        const isSelf = item.id === userId;
        // Only a superadmin may manage administrator / superadmin accounts.
        // An admin can still manage their own account and lower roles.
        const privileged = item.role === 'admin' || item.role === 'superadmin';
        const manageable = isSuperadmin || isSelf || !privileged;
        return (
          <div className="flex gap-1" onClick={(e) => e.stopPropagation()}>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              title={manageable ? 'Редактировать' : 'Управлять может только Супер администратор'}
              disabled={!manageable}
              onClick={() => openEdit(item)}
            >
              <Pencil className="h-3 w-3" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              title="Сбросить пароль"
              disabled={!manageable}
              onClick={() => {
                setResetTarget(item);
                setResetPassword('');
                setResetError(null);
              }}
            >
              <KeyRound className="h-3 w-3" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7 text-destructive"
              title={isSelf ? 'Нельзя удалить свою учётную запись' : !manageable ? 'Управлять может только Супер администратор' : 'Удалить'}
              disabled={isSelf || !manageable}
              onClick={() => setDeleteTarget(item)}
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
        );
      },
    },
  ];

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message="Не удалось загрузить пользователей." />;

  const editingSelf = editingUser?.id === userId;
  const submitting = createUser.isPending || updateUser.isPending;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Пользователи</h1>
        <Button onClick={openCreate}>
          <Plus className="h-4 w-4 mr-2" />
          Добавить пользователя
        </Button>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <SearchInput
          className="w-72"
          placeholder="Поиск по e-mail..."
          minChars={1}
          onSearch={(v) => {
            setSearch(v);
            setPage(1);
          }}
        />
        <Select
          value={roleFilter}
          onValueChange={(v) => {
            setRoleFilter((v as Role | 'all') || 'all');
            setPage(1);
          }}
        >
          <SelectTrigger className="w-52">
            <SelectValue>{roleFilter === 'all' ? 'Все роли' : roleLabel(roleFilter)}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Все роли</SelectItem>
            {ALL_ROLES.map((r) => (
              <SelectItem key={r} value={r}>
                {ROLE_LABELS[r]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" onClick={() => setSortDesc((s) => !s)} title="Сортировка по дате создания">
          <ArrowUpDown className="h-4 w-4 mr-2" />
          {sortDesc ? 'Сначала новые' : 'Сначала старые'}
        </Button>
      </div>

      <DataTable
        columns={columns}
        data={data?.data ?? []}
        total={data?.total ?? 0}
        page={page}
        limit={PAGE_SIZE}
        onPageChange={setPage}
      />

      {/* Create / edit dialog */}
      <Dialog open={showForm} onOpenChange={setShowForm}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingUser ? 'Редактировать пользователя' : 'Создать пользователя'}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">E-mail</Label>
              <Input
                id="email"
                type="email"
                value={formEmail}
                onChange={(e) => setFormEmail(e.target.value)}
                required
              />
            </div>

            {!editingUser && (
              <div className="space-y-2">
                <Label htmlFor="password">Начальный пароль</Label>
                <div className="flex gap-2">
                  <Input
                    id="password"
                    type="text"
                    value={formPassword}
                    onChange={(e) => setFormPassword(e.target.value)}
                    required
                    minLength={MIN_PASSWORD}
                    autoComplete="new-password"
                  />
                  <Button type="button" variant="outline" onClick={() => setFormPassword(generatePassword())}>
                    <RefreshCw className="h-4 w-4 mr-1" />
                    Сгенерировать
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground">Не менее {MIN_PASSWORD} символов.</p>
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="role">Роль</Label>
              <Select
                value={formRole}
                onValueChange={(v) => v && setFormRole(v as Role)}
                disabled={
                  editingSelf ||
                  (!!editingUser &&
                    (editingUser.role === 'admin' || editingUser.role === 'superadmin') &&
                    !isSuperadmin)
                }
              >
                <SelectTrigger>
                  <SelectValue>{roleLabel(formRole)}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {roleOptions.map((r) => (
                    <SelectItem key={r} value={r}>
                      {ROLE_LABELS[r]}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {editingSelf && (
                <p className="text-xs text-muted-foreground">Нельзя изменить собственную роль.</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="status">Статус</Label>
              <Select
                value={formActive ? 'active' : 'inactive'}
                onValueChange={(v) => setFormActive(v === 'active')}
                disabled={editingSelf}
              >
                <SelectTrigger>
                  <SelectValue>{formActive ? 'Активен' : 'Неактивен'}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="active">Активен</SelectItem>
                  <SelectItem value="inactive">Неактивен</SelectItem>
                </SelectContent>
              </Select>
              {editingSelf && (
                <p className="text-xs text-muted-foreground">Нельзя деактивировать собственную учётную запись.</p>
              )}
            </div>

            {formError && <p className="text-sm text-destructive">{formError}</p>}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setShowForm(false)}>
                Отмена
              </Button>
              <Button type="submit" disabled={submitting}>
                {editingUser ? 'Сохранить' : 'Создать'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Reset password dialog */}
      <Dialog
        open={!!resetTarget}
        onOpenChange={(open) => {
          if (!open) {
            setResetTarget(null);
            setResetPassword('');
            setResetError(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Сбросить пароль</DialogTitle>
            <DialogDescription>
              Задайте новый пароль для пользователя {resetTarget?.email}. Текущий пароль перестанет действовать.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div className="flex gap-2">
              <Input
                type="text"
                value={resetPassword}
                placeholder="Новый пароль"
                onChange={(e) => setResetPassword(e.target.value)}
                minLength={MIN_PASSWORD}
                autoComplete="new-password"
              />
              <Button type="button" variant="outline" onClick={() => setResetPassword(generatePassword())}>
                <RefreshCw className="h-4 w-4 mr-1" />
                Сгенерировать
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">Не менее {MIN_PASSWORD} символов.</p>
            {resetError && <p className="text-sm text-destructive">{resetError}</p>}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setResetTarget(null);
                setResetPassword('');
                setResetError(null);
              }}
            >
              Отмена
            </Button>
            <Button variant="destructive" onClick={handleResetPassword} disabled={updateUser.isPending}>
              Сбросить пароль
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete confirm */}
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title="Удалить пользователя"
        description={`Учётная запись ${deleteTarget?.email ?? ''} будет удалена без возможности восстановления. Продолжить?`}
        onConfirm={handleDelete}
        loading={deleteUser.isPending}
      />

      {/* Superadmin role-assignment confirm */}
      <ConfirmDialog
        open={pendingSubmit}
        onOpenChange={(open) => !open && setPendingSubmit(false)}
        title="Назначить роль «Супер администратор»"
        description="Супер администратор получает полный доступ ко всем данным, пользователям и системным настройкам. Подтвердите назначение этой роли."
        onConfirm={doSubmit}
        loading={submitting}
        destructive={false}
      />
    </div>
  );
}
