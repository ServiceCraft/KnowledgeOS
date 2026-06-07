import type { Role } from '@/types';

const ROLE_LEVELS: Record<string, number> = {
  viewer: 0,
  editor: 1,
  admin: 2,
  superadmin: 3,
};

// Localized role names per the platform role dictionary (TZ §4.2).
export const ROLE_LABELS: Record<Role, string> = {
  viewer: 'Оператор',
  editor: 'Менеджер',
  admin: 'Администратор',
  superadmin: 'Супер администратор',
};

// Ordered list of roles for selects (lowest privilege first).
export const ALL_ROLES: Role[] = ['viewer', 'editor', 'admin', 'superadmin'];

export function roleLabel(role: string): string {
  return ROLE_LABELS[role as Role] ?? role;
}

export function hasMinRole(userRole: string, minRole: string): boolean {
  return (ROLE_LEVELS[userRole] ?? -1) >= (ROLE_LEVELS[minRole] ?? 999);
}

// Roles that may create/edit/delete business entities (editor and above).
export function canWrite(role: string | undefined): boolean {
  return hasMinRole(role ?? 'viewer', 'editor');
}

// Roles that may access the Users section (admin and above).
export function canManageUsers(role: string | undefined): boolean {
  return hasMinRole(role ?? 'viewer', 'admin');
}

export function isSuperadmin(role: string | undefined): boolean {
  return role === 'superadmin';
}
