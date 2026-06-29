import { useNavigate, useLocation } from 'react-router-dom';
import { Building2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { LoadingState } from '@/components/shared/LoadingState';
import { ErrorState } from '@/components/shared/ErrorState';
import { useAccessibleCompanies } from '@/hooks/useAccessibleCompanies';
import { useSelectCompanyContext } from '@/hooks/useCompanyContext';

export function SelectCompanyPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { data: companies, isLoading, isError } = useAccessibleCompanies();
  const selectCompany = useSelectCompanyContext();

  const from = (location.state as { from?: string } | null)?.from ?? '/kb';

  if (isLoading) return <LoadingState />;
  if (isError) return <ErrorState message="Не удалось загрузить список компаний." />;

  return (
    <div className="mx-auto flex max-w-lg flex-col gap-6 p-8">
      <div className="space-y-2 text-center">
        <Building2 className="mx-auto h-10 w-10 text-muted-foreground" />
        <h1 className="text-2xl font-semibold">Выберите компанию</h1>
        <p className="text-sm text-muted-foreground">
          У вашей учётной записи есть доступ к нескольким компаниям. Выберите, с какой работать.
        </p>
      </div>

      <div className="flex flex-col gap-2">
        {(companies ?? []).map((company) => (
          <Button
            key={company.id}
            variant="outline"
            className="h-auto justify-start px-4 py-3"
            onClick={() => {
              selectCompany(company.id, company.name);
              navigate(from, { replace: true });
            }}
          >
            <span className="font-medium">{company.name}</span>
          </Button>
        ))}
      </div>
    </div>
  );
}
