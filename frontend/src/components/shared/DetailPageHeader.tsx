import { Link } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { Button } from '@/components/ui/button';

interface DetailPageHeaderProps {
  backTo: string;
  backLabel?: string;
  title: string;
  description?: string;
  actions?: React.ReactNode;
}

export function DetailPageHeader({
  backTo,
  backLabel = 'Назад',
  title,
  description,
  actions,
}: DetailPageHeaderProps) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div className="space-y-2">
        <Button variant="ghost" size="sm" render={<Link to={backTo} />}>
          <ArrowLeft className="h-4 w-4 mr-1" />
          {backLabel}
        </Button>
        <div>
          <h1 className="text-2xl font-semibold">{title}</h1>
          {description && <p className="text-sm text-muted-foreground">{description}</p>}
        </div>
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </div>
  );
}
