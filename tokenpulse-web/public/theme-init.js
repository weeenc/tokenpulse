// Run before styles load; an external script also works with script-src 'self'.
(() => {
  let theme;
  try {
    theme = window.localStorage.getItem('tokenpulse-theme');
  } catch {
    // The system preference remains available when storage is blocked.
  }
  if (theme !== 'light' && theme !== 'dark') {
    theme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  document.documentElement.dataset.theme = theme;
  document.querySelector('meta[name="theme-color"]').content =
    theme === 'dark' ? '#050506' : '#f5f6fb';
})();
