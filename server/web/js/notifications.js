const notifyBtn = document.getElementById('notify-btn');

function updateNotifyBtn(enabled) {
  notifyBtn.querySelector('.material-icons').textContent = enabled ? 'notifications' : 'notifications_off';
  notifyBtn.setAttribute('aria-label', enabled ? 'Disable notifications' : 'Enable notifications');
  notifyBtn.classList.toggle('disabled', !enabled);
}

async function loadNotificationsState() {
  try {
    const res = await fetch('/api/notifications');
    const data = await res.json();
    updateNotifyBtn(data.enabled);
  } catch {}
}

export async function toggleNotifications() {
  try {
    const res = await fetch('/api/notifications', { method: 'POST' });
    const data = await res.json();
    updateNotifyBtn(data.enabled);
  } catch {}
}

export function init() {
  loadNotificationsState();
}
