import { mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import ThemeToggle from '../src/components/ThemeToggle.vue';
import { initializeTheme, useTheme } from '../src/utils/theme.js';

describe('page theme', () => {
  let systemTheme: EventTarget & { matches: boolean };
  let cleanup: () => void;

  beforeEach(() => {
    const storage = new Map<string, string>();
    vi.stubGlobal('localStorage', {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      clear: () => storage.clear(),
    });
    document.head.innerHTML = '<meta name="theme-color" content="#050506">';
    systemTheme = Object.assign(new EventTarget(), { matches: false });
    vi.stubGlobal(
      'matchMedia',
      vi.fn(() => systemTheme),
    );
  });

  afterEach(() => {
    cleanup?.();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
    delete document.documentElement.dataset.theme;
  });

  function changeSystemTheme(dark: boolean): void {
    systemTheme.matches = dark;
    systemTheme.dispatchEvent(new Event('change'));
  }

  it('follows system changes until the user chooses a theme', () => {
    cleanup = initializeTheme();
    expect(document.documentElement.dataset.theme).toBe('light');
    changeSystemTheme(true);
    expect(document.documentElement.dataset.theme).toBe('dark');
    useTheme().toggleTheme();
    changeSystemTheme(false);
    changeSystemTheme(true);
    expect(document.documentElement.dataset.theme).toBe('light');
    expect(window.localStorage.getItem('tokenpulse-theme')).toBe('light');
  });

  it('restores the explicit choice even when the system theme differs', () => {
    window.localStorage.setItem('tokenpulse-theme', 'dark');
    cleanup = initializeTheme();
    expect(useTheme().theme.value).toBe('dark');
    expect(document.documentElement.dataset.theme).toBe('dark');
    expect(document.querySelector('meta[name="theme-color"]')?.getAttribute('content')).toBe(
      '#050506',
    );
  });

  it('ignores invalid preferences and synchronizes changes from another tab', () => {
    window.localStorage.setItem('tokenpulse-theme', 'invalid');
    cleanup = initializeTheme();
    expect(useTheme().theme.value).toBe('light');
    window.localStorage.setItem('tokenpulse-theme', 'dark');
    window.dispatchEvent(new StorageEvent('storage', { key: 'tokenpulse-theme' }));
    expect(document.documentElement.dataset.theme).toBe('dark');
    window.localStorage.clear();
    window.dispatchEvent(new StorageEvent('storage', { key: null }));
    expect(document.documentElement.dataset.theme).toBe('light');
  });

  it('switches the page and keeps both toggle labels in sync', async () => {
    window.localStorage.setItem('tokenpulse-theme', 'dark');
    cleanup = initializeTheme();
    const desktop = mount(ThemeToggle);
    const mobile = mount(ThemeToggle);
    await desktop.get('button').trigger('click');
    expect(document.documentElement.dataset.theme).toBe('light');
    expect(document.querySelector('meta[name="theme-color"]')?.getAttribute('content')).toBe(
      '#f5f6fb',
    );
    expect(desktop.get('button').attributes('aria-label')).toBe('切换到深色模式');
    expect(mobile.get('button').attributes('aria-label')).toBe('切换到深色模式');
    await mobile.get('button').trigger('click');
    expect(desktop.get('button').attributes('aria-label')).toBe('切换到亮色模式');
    desktop.unmount();
    mobile.unmount();
  });

  it('still switches when local storage is unavailable', () => {
    vi.spyOn(window.localStorage, 'getItem').mockImplementation(() => {
      throw new Error('Blocked');
    });
    vi.spyOn(window.localStorage, 'setItem').mockImplementation(() => {
      throw new Error('Blocked');
    });
    cleanup = initializeTheme();
    expect(() => useTheme().toggleTheme()).not.toThrow();
    expect(document.documentElement.dataset.theme).toBe('dark');
  });

  it('removes listeners when the application unmounts', () => {
    cleanup = initializeTheme();
    cleanup();
    changeSystemTheme(true);
    expect(document.documentElement.dataset.theme).toBe('light');
  });
});
