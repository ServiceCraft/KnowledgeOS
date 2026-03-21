import { useLocation } from 'react-router-dom';
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
import { Moon, Sun, LogOut } from 'lucide-react';
import { useAuthStore } from '@/stores/authStore';
import { useUIStore } from '@/stores/uiStore';
import { useLogout } from '@/hooks/useAuth';

const routeLabels: Record<string, string> = {
  kb: 'База знаний',
  qa: 'Вопросы и ответы',
  themes: 'Темы',
  articles: 'Статьи',
  faq: 'FAQ',
  pricing: 'Прайс',
  search: 'Поиск',
  settings: 'Настройки',
  users: 'Пользователи',
  sync: 'Синхронизация',
  export: 'Экспорт',
  admin: 'Администрирование',
  companies: 'Компании',
};

export function Header() {
  const location = useLocation();
  const user = useAuthStore((s) => s.user);
  const { darkMode, toggleDarkMode } = useUIStore();
  const logout = useLogout();

  const segments = location.pathname.split('/').filter(Boolean);
  const breadcrumbs = segments.map((seg, i) => ({
    label: routeLabels[seg] || seg,
    path: '/' + segments.slice(0, i + 1).join('/'),
    isLast: i === segments.length - 1,
  }));

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
