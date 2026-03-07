import { toggleMute, toggleFullscreen } from './controls.js';
import { init as initDetection } from './detection.js';
import { openLogModal, closeLogModal } from './logs.js';
import { init as initNotifications, toggleNotifications } from './notifications.js';
import { init as initStatus } from './status.js';
import { init as initTheme } from './theme.js';
import { start } from './webrtc.js';
import { init as initWaveform } from './waveform.js';

const el = (id: string): HTMLElement => document.getElementById(id) as HTMLElement;

// Wire up UI event listeners
el('overlayMuteBtn').addEventListener('click', toggleMute);
el('overlayFsBtn').addEventListener('click', toggleFullscreen);
el('notify-btn').addEventListener('click', toggleNotifications);
el('log-btn').addEventListener('click', openLogModal);
el('status-btn').addEventListener('click', openLogModal);
el('log-close-btn').addEventListener('click', closeLogModal);
el('log-modal').addEventListener('click', closeLogModal);
el('log-card').addEventListener('click', e => e.stopPropagation());

// Initialize modules
initDetection();
initNotifications();
initStatus();
initTheme();
initWaveform();

// not just init, as this is re-invoked on each reconnect
start().catch(err => {
  const streamErr = document.getElementById('stream-error');
  if (streamErr) { streamErr.textContent = `Stream not available: ${(err as Error).message}`; streamErr.classList.add('visible'); }
});
