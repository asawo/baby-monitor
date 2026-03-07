import type { NotificationsResponse } from './api_types.js';

const notifyBtn = document.getElementById('notify-btn') as HTMLElement;

function updateNotifyBtn(enabled: boolean) {
  (notifyBtn.querySelector('.material-icons') as HTMLElement).textContent = enabled ? 'notifications' : 'notifications_off';
  notifyBtn.setAttribute('aria-label', enabled ? 'Disable notifications' : 'Enable notifications');
  notifyBtn.classList.toggle('disabled', !enabled);
}

async function loadNotificationsState() {
  try {
    const res = await fetch('/api/notifications');
    const data: NotificationsResponse = await res.json();
    updateNotifyBtn(data.enabled);
  } catch {}
}

export async function toggleNotifications() {
  try {
    const res = await fetch('/api/notifications', { method: 'POST' });
    const data: NotificationsResponse = await res.json();
    updateNotifyBtn(data.enabled);
  } catch {}
}

export function init() {
  loadNotificationsState();
}
