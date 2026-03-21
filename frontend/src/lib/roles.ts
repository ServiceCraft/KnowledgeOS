const ROLE_LEVELS: Record<string, number> = {
  viewer: 0,
  editor: 1,
  admin: 2,
  superadmin: 3,
};

export function hasMinRole(userRole: string, minRole: string): boolean {
  return (ROLE_LEVELS[userRole] ?? -1) >= (ROLE_LEVELS[minRole] ?? 999);
}
