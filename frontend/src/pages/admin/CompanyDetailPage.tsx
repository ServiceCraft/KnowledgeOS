import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { ArrowLeft, Plus } from 'lucide-react';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { useCompanyDetail, useCreateCompanyAdmin } from '@/hooks/useAdmin';
import { useSelectCompanyContext } from '@/hooks/useCompanyContext';
import { useAuthStore } from '@/stores/authStore';
import { toast } from 'sonner';

export function CompanyDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: company, isLoading, isError } = useCompanyDetail(id!);
  const createAdmin = useCreateCompanyAdmin(id!);
  const selectedCompanyId = useAuthStore((s) => s.selectedCompanyId);
  const selectCompany = useSelectCompanyContext();

  const [showAdmin, setShowAdmin] = useState(false);
  const [adminEmail, setAdminEmail] = useState('');
  const [adminPassword, setAdminPassword] = useState('');

  const handleCreateAdmin = (e: React.FormEvent) => {
    e.preventDefault();
    createAdmin.mutate(
      { email: adminEmail, password: adminPassword },
      {
        onSuccess: () => {
          setShowAdmin(false);
          setAdminEmail('');
          setAdminPassword('');
          toast.success('Администратор создан');
        },
        onError: () => toast.error('Не удалось создать администратора'),
      }
    );
  };

  if (isLoading) return <LoadingState />;
  if (isError || !company) return <ErrorState message="Компания не найдена." />;

  return (
    <div className="space-y-6 max-w-2xl">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate('/admin/companies')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-2xl font-semibold flex-1">{company.name}</h1>
        <Button
          variant={selectedCompanyId === company.id ? 'default' : 'outline'}
          onClick={() => {
            selectCompany(company.id, company.name);
            toast.success(`Выбрана компания: ${company.name}`);
            navigate('/kb');
          }}
        >
          {selectedCompanyId === company.id ? 'Компания выбрана' : 'Выбрать для работы'}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Информация</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <p><span className="text-muted-foreground">ID:</span> {company.id}</p>
          <p>
            <span className="text-muted-foreground">Тариф:</span>{' '}
            <Badge variant="secondary">{company.tier}</Badge>
          </p>
          <p><span className="text-muted-foreground">Создана:</span> {new Date(company.created_at).toLocaleString()}</p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">Администратор</CardTitle>
          <Button size="sm" variant="outline" onClick={() => setShowAdmin(true)}>
            <Plus className="h-4 w-4 mr-1" />
            Добавить админа
          </Button>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Создайте администратора для этой компании (отдельный вызов API).
          </p>
        </CardContent>
      </Card>

      <Dialog open={showAdmin} onOpenChange={setShowAdmin}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Новый администратор</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreateAdmin} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="admin-email">Email</Label>
              <Input
                id="admin-email"
                type="email"
                value={adminEmail}
                onChange={(e) => setAdminEmail(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="admin-password">Пароль</Label>
              <Input
                id="admin-password"
                type="password"
                value={adminPassword}
                onChange={(e) => setAdminPassword(e.target.value)}
                required
                minLength={8}
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setShowAdmin(false)}>Отмена</Button>
              <Button type="submit" disabled={createAdmin.isPending}>Создать</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
