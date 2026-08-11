export interface NavigationTarget {
  path: string;
  fullPath: string;
  isPublic: boolean;
  returnUrl?: unknown;
}

export function authNavigation(
  target: NavigationTarget,
  authenticated: boolean,
): true | string | { path: string; query: { returnUrl: string } } {
  if (!target.isPublic && !authenticated) {
    return { path: '/login', query: { returnUrl: target.fullPath } };
  }
  if (target.isPublic && authenticated && ['/login', '/register'].includes(target.path)) {
    return typeof target.returnUrl === 'string' &&
      target.returnUrl.startsWith('/') &&
      !target.returnUrl.startsWith('//')
      ? target.returnUrl
      : '/dashboard';
  }
  return true;
}
