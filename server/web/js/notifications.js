/** @import { NotificationsResponse } from './api_types' */

const notifyBtn = /** @type {HTMLElement} */ (document.getElementById('notify-btn'));

/** @param {boolean} enabled */
function updateNotifyBtn(enabled) {
  /** @type {HTMLElement} */ (notifyBtn.querySelector('.material-icons')).textContent = enabled ? 'notifications' : 'notifications_off';
  notifyBtn.setAttribute('aria-label', enabled ? 'Disable notifications' : 'Enable notifications');
  notifyBtn.classList.toggle('disabled', !enabled);
}

async function loadNotificationsState() {
  try {
    const res = await fetch('/api/notifications');
    /** @type {NotificationsResponse} */
    const data = await res.json();
    updateNotifyBtn(data.enabled);
  } catch {}
}

export async function toggleNotifications() {
  try {
    const res = await fetch('/api/notifications', { method: 'POST' });
    /** @type {NotificationsResponse} */
    const data = await res.json();
    updateNotifyBtn(data.enabled);
  } catch {}
}

export function init() {
  loadNotificationsState();
}
