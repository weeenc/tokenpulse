import { readonly, ref } from 'vue';

type Theme = 'light' | 'dark';
const storageKey = 'tokenpulse-theme';
const theme = ref<Theme>('dark');
let preference: Theme | null = null;

function savedTheme(): Theme | null {
  try {
    const saved = window.localStorage.getItem(storageKey);
    return saved === 'light' || saved === 'dark' ? saved : null;
  } catch {
    return null;
  }
}

function applyTheme(value: Theme): void {
  document.documentElement.dataset.theme = value;
  document
    .querySelector('meta[name="theme-color"]')
    ?.setAttribute('content', value === 'dark' ? '#050506' : '#f5f6fb');
  theme.value = value;
}

export function initializeTheme(): () => void {
  const systemTheme = window.matchMedia('(prefers-color-scheme: dark)');
  preference = savedTheme();
  const syncTheme = () => applyTheme(preference ?? (systemTheme.matches ? 'dark' : 'light'));
  const syncStorage = (event: StorageEvent) => {
    if (event.key !== storageKey && event.key !== null) return;
    preference = savedTheme();
    syncTheme();
  };

  syncTheme();
  systemTheme.addEventListener('change', syncTheme);
  window.addEventListener('storage', syncStorage);
  return () => {
    systemTheme.removeEventListener('change', syncTheme);
    window.removeEventListener('storage', syncStorage);
  };
}

export function useTheme() {
  function toggleTheme(): void {
    preference = theme.value === 'dark' ? 'light' : 'dark';
    applyTheme(preference);
    try {
      window.localStorage.setItem(storageKey, preference);
    } catch {
      // Switching still works when the browser does not allow persistent storage.
    }
  }

  return { theme: readonly(theme), toggleTheme };
}
