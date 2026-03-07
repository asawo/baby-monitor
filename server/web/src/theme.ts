type Theme = 'light' | 'dark';

const STORAGE_KEY = 'monitor-theme';

function apply(theme: Theme) {
  document.documentElement.setAttribute('data-theme', theme);
  const btn = document.getElementById('theme-btn');
  if (btn) {
    const next = theme === 'dark' ? 'light' : 'dark';
    btn.innerHTML = `<span class="material-icons">${theme === 'dark' ? 'dark_mode' : 'light_mode'}</span>`;
    btn.setAttribute('aria-label', `Switch to ${next} mode`);
  }
}

export function init() {
  const saved = localStorage.getItem(STORAGE_KEY) as Theme | null;
  const theme = saved ?? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
  apply(theme);
  document.getElementById('theme-btn')?.addEventListener('click', () => {
    const current = document.documentElement.getAttribute('data-theme') as Theme;
    const next: Theme = current === 'dark' ? 'light' : 'dark';
    localStorage.setItem(STORAGE_KEY, next);
    apply(next);
  });
}
