import { Link, useLocation } from 'react-router-dom';
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from '@/components/ui/breadcrumb';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { SidebarTrigger } from '@/components/ui/sidebar';
import { Separator } from '@/components/ui/separator';
import { Moon, Sun, LogOut, Building2, AlertCircle } from 'lucide-react';
import { useAuthStore } from '@/stores/authStore';
import { useUIStore } from '@/stores/uiStore';
import { useLogout } from '@/hooks/useAuth';
import { useAccessibleCompanies } from '@/hooks/useAccessibleCompanies';
import { cn } from '@/lib/utils';

const routeLabels: Record<string, string> = {
  kb: 'База знаний',
  qa: 'Вопросы и ответы',
  themes: 'Темы',
  articles: 'Статьи',
  new: 'Новая статья',
  faq: 'FAQ',
  pricing: 'Прайс',
  search: 'Поиск',
  settings: 'Настройки',
  bot: 'Бот',
  playground: 'Плейграунд',
  handoff: 'Диалоги / Эскалации',
  users: 'Пользователи',
  sync: 'Синхронизация',
  export: 'Экспорт',
  admin: 'Администрирование',
  companies: 'Компании',
};

// Названия для страниц, где последний сегмент сам по себе неинформативен
// (крошки при вложенности < 3 схлопываются до одного заголовка).
const pathLabels: Record<string, string> = {
  '/settings/bot': 'Настройки бота',
  '/settings/export': 'Экспорт / Импорт',
};

export function Header() {
  const location = useLocation();
  const user = useAuthStore((s) => s.user);
  const selectedCompanyId = useAuthStore((s) => s.selectedCompanyId);
  const selectedCompanyName = useAuthStore((s) => s.selectedCompanyName);
  const { darkMode, toggleDarkMode } = useUIStore();
  const logout = useLogout();
  const hasMultipleCompanies = (user?.company_ids?.length ?? 0) > 1;
  const showCompanyPicker = user?.role === 'superadmin' || hasMultipleCompanies;
  const needsCompanyList =
    showCompanyPicker || (!!selectedCompanyId && !selectedCompanyName);
  const { data: accessibleCompanies } = useAccessibleCompanies({ enabled: needsCompanyList });
  const companyPickerHref = user?.role === 'superadmin' ? '/admin/companies' : '/select-company';

  const resolvedCompanyName =
    selectedCompanyName ??
    accessibleCompanies?.find((c) => c.id === selectedCompanyId)?.name ??
    selectedCompanyId;

  const segments = location.pathname.split('/').filter(Boolean);
  const allCrumbs = segments.map((seg, i) => {
    const path = '/' + segments.slice(0, i + 1).join('/');
    return {
      label: pathLabels[path] || routeLabels[seg] || seg,
      path,
      isLast: i === segments.length - 1,
    };
  });
  // При вложенности < 3 цепочка не нужна: промежуточных страниц нет,
  // ссылка на «родителя» вела бы на редирект (/settings → /settings/users).
  // На глубоких путях первый сегмент (/kb, /admin) — тоже редирект, убираем его.
  const breadcrumbs = allCrumbs.length >= 3 ? allCrumbs.slice(1) : allCrumbs.slice(-1);

  const initials = user?.email
    ?.split('@')[0]
    .slice(0, 2)
    .toUpperCase() ?? '??';

  return (
    <header className="flex items-center gap-2 border-b px-4 h-14 shrink-0">
      <SidebarTrigger />
      <Separator orientation="vertical" className="h-5" />
      <Breadcrumb className="flex-1">
        <BreadcrumbList>
          {breadcrumbs.map((crumb, i) => (
            <span key={crumb.path} className="contents">
              {i > 0 && <BreadcrumbSeparator />}
              <BreadcrumbItem>
                {crumb.isLast ? (
                  <BreadcrumbPage>{crumb.label}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink href={crumb.path}>{crumb.label}</BreadcrumbLink>
                )}
              </BreadcrumbItem>
            </span>
          ))}
        </BreadcrumbList>
      </Breadcrumb>

      {showCompanyPicker && (
        <Link
          to={companyPickerHref}
          className={cn(
            'flex min-w-0 max-w-[min(100%,18rem)] shrink items-center gap-2 rounded-lg border px-3 py-1.5 text-sm transition',
            selectedCompanyId
              ? 'border-primary bg-primary/10 font-semibold text-primary shadow-sm ring-1 ring-primary/20 hover:bg-primary/15'
              : 'animate-pulse border-destructive/60 bg-destructive/10 font-medium text-destructive hover:bg-destructive/15'
          )}
          title={selectedCompanyId ? String(resolvedCompanyName) : 'Компания не выбрана — нажмите, чтобы выбрать'}
        >
          {selectedCompanyId ? (
            <Building2 className="h-4 w-4 shrink-0" aria-hidden />
          ) : (
            <AlertCircle className="h-4 w-4 shrink-0" aria-hidden />
          )}
          <span className="truncate">
            {selectedCompanyId ? (
              <>
                <span className="hidden sm:inline">Компания: </span>
                {resolvedCompanyName}
              </>
            ) : (
              'Выберите компанию'
            )}
          </span>
        </Link>
      )}

      <DropdownMenu>
        <DropdownMenuTrigger className="flex items-center gap-2 rounded-full focus:outline-none focus:ring-2 focus:ring-ring">
          <Avatar className="h-8 w-8">
            <AvatarFallback className="text-xs">{initials}</AvatarFallback>
          </Avatar>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <div className="px-2 py-1.5 text-sm">
            <p className="font-medium">{user?.email}</p>
            <p className="text-xs text-muted-foreground capitalize">{user?.role}</p>
          </div>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={toggleDarkMode}>
            {darkMode ? <Sun className="h-4 w-4 mr-2" /> : <Moon className="h-4 w-4 mr-2" />}
            {darkMode ? 'Светлая тема' : 'Тёмная тема'}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => logout.mutate()} className="text-destructive">
            <LogOut className="h-4 w-4 mr-2" />
            Выйти
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </header>
  );
}
