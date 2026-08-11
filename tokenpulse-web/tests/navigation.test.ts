import { describe, expect, it } from 'vitest';
import { authNavigation } from '../src/utils/navigation.js';

describe('authentication navigation', () => {
  it('preserves the complete device authorization URL', () => {
    expect(
      authNavigation(
        {
          path: '/device',
          fullPath: '/device?code=ABCD-EFGH',
          isPublic: false,
        },
        false,
      ),
    ).toEqual({ path: '/login', query: { returnUrl: '/device?code=ABCD-EFGH' } });
  });

  it('returns an authenticated user to a safe local URL', () => {
    expect(
      authNavigation(
        { path: '/login', fullPath: '/login', isPublic: true, returnUrl: '/devices' },
        true,
      ),
    ).toBe('/devices');
    expect(
      authNavigation(
        { path: '/login', fullPath: '/login', isPublic: true, returnUrl: 'https://evil.test' },
        true,
      ),
    ).toBe('/dashboard');
    expect(
      authNavigation(
        { path: '/login', fullPath: '/login', isPublic: true, returnUrl: '//evil.test' },
        true,
      ),
    ).toBe('/dashboard');
  });
});
