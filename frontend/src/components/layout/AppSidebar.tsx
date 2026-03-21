import { useLocation, Link } from 'react-router-dom';
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarFooter,
} from '@/components/ui/sidebar';
import {
  HelpCircle,
  BookOpen,
  Tags,
  FileText,
  DollarSign,
  Search,
  // Users,
  RefreshCw,
  Download,
  Building2,
  MessageSquareQuote,
} from 'lucide-react';
import { useAuthStore } from '@/stores/authStore';
import { hasMinRole } from '@/lib/roles';

const kbLinks = [
  { to: '/kb/qa', label: 'Вопросы и ответы', icon: HelpCircle },
  { to: '/kb/themes', label: 'Темы', icon: Tags },
  { to: '/kb/articles', label: 'Статьи', icon: FileText },
  { to: '/kb/faq', label: 'FAQ', icon: MessageSquareQuote },
  { to: '/kb/pricing', label: 'Прайс', icon: DollarSign },
];

const toolLinks = [
  { to: '/kb/search', label: 'Поиск', icon: Search },
];

const settingsLinks = [
  // { to: '/settings/users', label: 'Пользователи', icon: Users, minRole: 'admin' as const },
  { to: '/settings/sync', label: 'Статус синхронизации', icon: RefreshCw, minRole: 'admin' as const },
  { to: '/settings/export', label: 'Экспорт / Импорт', icon: Download, minRole: 'admin' as const },
];

const adminLinks = [
  { to: '/admin/companies', label: 'Компании', icon: Building2, minRole: 'superadmin' as const },
];

export function AppSidebar() {
  const location = useLocation();
  const user = useAuthStore((s) => s.user);
  const role = user?.role ?? 'viewer';

  const isActive = (path: string) => location.pathname === path || location.pathname.startsWith(path + '/');

  return (
    <Sidebar>
      <SidebarHeader className="p-4">
        <Link to="/kb" className="flex items-center gap-2">
          <BookOpen className="h-6 w-6 text-primary" />
          <span className="font-semibold text-lg">KnowledgeOS</span>
        </Link>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>База знаний</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {kbLinks.map((link) => (
                <SidebarMenuItem key={link.to}>
                  <SidebarMenuButton render={<Link to={link.to} />} isActive={isActive(link.to)}>
                    <link.icon className="h-4 w-4" />
                    <span>{link.label}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup>
          <SidebarGroupLabel>Инструменты</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {toolLinks.map((link) => (
                <SidebarMenuItem key={link.to}>
                  <SidebarMenuButton render={<Link to={link.to} />} isActive={isActive(link.to)}>
                    <link.icon className="h-4 w-4" />
                    <span>{link.label}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {hasMinRole(role, 'admin') && (
          <SidebarGroup>
            <SidebarGroupLabel>Настройки</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {settingsLinks
                  .filter((link) => hasMinRole(role, link.minRole))
                  .map((link) => (
                    <SidebarMenuItem key={link.to}>
                      <SidebarMenuButton render={<Link to={link.to} />} isActive={isActive(link.to)}>
                        <link.icon className="h-4 w-4" />
                        <span>{link.label}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}

        {hasMinRole(role, 'superadmin') && (
          <SidebarGroup>
            <SidebarGroupLabel>Администрирование</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {adminLinks.map((link) => (
                  <SidebarMenuItem key={link.to}>
                    <SidebarMenuButton render={<Link to={link.to} />} isActive={isActive(link.to)}>
                      <link.icon className="h-4 w-4" />
                      <span>{link.label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        )}
      </SidebarContent>
      <SidebarFooter className="p-4">
        <p className="text-xs text-muted-foreground">{user?.email}</p>
      </SidebarFooter>
    </Sidebar>
  );
}
