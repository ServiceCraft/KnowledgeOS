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
  FileText,
  DollarSign,
  Layers,
  Search,
  Users,
  Download,
  MessageSquareQuote,
  Headphones,
  Bot,
  Settings,
  Building2,
} from 'lucide-react';
import { useAuthStore } from '@/stores/authStore';
import { isTenantReady } from '@/lib/tenantContext';
import { hasMinRole } from '@/lib/roles';

const kbLinks = [
  { to: '/kb/themes', label: 'Темы', icon: Layers },
  { to: '/kb/qa', label: 'Вопросы и ответы', icon: HelpCircle },
  { to: '/kb/articles', label: 'Статьи', icon: FileText },
  { to: '/kb/faq', label: 'FAQ', icon: MessageSquareQuote },
  { to: '/kb/pricing', label: 'Прайс', icon: DollarSign },
];

const toolLinks = [
  { to: '/kb/search', label: 'Поиск', icon: Search },
];

const botLinks = [
  { to: '/bot/playground', label: 'Плейграунд', icon: Bot },
  { to: '/bot/handoff', label: 'Диалоги / Эскалации', icon: Headphones },
  { to: '/settings/bot', label: 'Настройки бота', icon: Settings, minRole: 'admin' as const },
];

const settingsLinks = [
  { to: '/settings/users', label: 'Пользователи', icon: Users, minRole: 'admin' as const },
  { to: '/admin/companies', label: 'Компании', icon: Building2, minRole: 'superadmin' as const },
  { to: '/settings/export', label: 'Экспорт / Импорт', icon: Download, minRole: 'superadmin' as const },
];

export function AppSidebar() {
  const location = useLocation();
  const user = useAuthStore((s) => s.user);
  const selectedCompanyId = useAuthStore((s) => s.selectedCompanyId);
  const role = user?.role ?? 'viewer';
  const tenantReady = isTenantReady(user, selectedCompanyId);

  const isActive = (path: string) => location.pathname === path || location.pathname.startsWith(path + '/');
  const visibleSettingsLinks = settingsLinks.filter((link) => {
    if (!hasMinRole(role, link.minRole)) return false;
    if (tenantReady) return true;
    return link.to.startsWith('/admin/companies');
  });

  return (
    <Sidebar>
      <SidebarHeader className="p-4">
        <Link to={tenantReady ? '/kb' : '/admin/companies'} className="flex items-center gap-2">
          <BookOpen className="h-6 w-6 text-primary" />
          <span className="font-semibold text-lg">KnowledgeOS</span>
        </Link>
      </SidebarHeader>
      <SidebarContent>
        {tenantReady && (
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
        )}

        {tenantReady && (
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
        )}

        {tenantReady && (
          <SidebarGroup>
            <SidebarGroupLabel>Бот</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {botLinks
                  .filter((link) => !('minRole' in link) || hasMinRole(role, link.minRole!))
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

        {visibleSettingsLinks.length > 0 && (
          <SidebarGroup>
            <SidebarGroupLabel>Настройки</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {visibleSettingsLinks.map((link) => (
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
