const LOG_SECTION_COLORS = {
  'mediamtx':             '#60a5fa', // blue
  'stream.service':       '#34d399', // green
  'monitor-http.service': '#f472b6', // pink
  'detect.service':       '#fb923c', // orange
};
const LOG_SECTION_COLOR_FALLBACK = '#aaa';
const LOG_MAX_LINES = 30;

export async function openLogModal() {
  document.getElementById('log-modal').classList.add('open');
  const container = document.getElementById('log-content');
  container.innerHTML = '<span style="color:#888">Loading…</span>';
  try {
    const res = await fetch('/api/logs');
    const sections = await res.json();
    container.innerHTML = '';
    sections.forEach(s => {
      const color = LOG_SECTION_COLORS[s.name] ?? LOG_SECTION_COLOR_FALLBACK;
      const lines = s.content.trimEnd().split('\n');
      const truncated = lines.slice(-LOG_MAX_LINES);
      const div = document.createElement('div');
      div.className = 'log-section open';
      div.innerHTML = `
        <div class="log-section-header" style="color:${color}">
          <span>${s.name}</span>
          <span class="material-icons">chevron_right</span>
        </div>
        <div class="log-section-body">${truncated.join('\n').replace(/</g, '&lt;')}</div>`;
      div.querySelector('.log-section-header').addEventListener('click', () => {
        div.classList.toggle('open');
      });
      container.appendChild(div);
    });
  } catch (e) {
    container.innerHTML = '<span style="color:#f87171">Failed to load logs: ' + e.message + '</span>';
  }
}

export function closeLogModal() {
  document.getElementById('log-modal').classList.remove('open');
}

document.addEventListener('keydown', e => {
  if (e.key === 'Escape') closeLogModal();
});
