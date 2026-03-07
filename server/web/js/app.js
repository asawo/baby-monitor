import { init as initWaveform } from './waveform.js';
import { init as initStatus } from './status.js';
import { init as initNotifications, toggleNotifications } from './notifications.js';
import { init as initDetection } from './detection.js';
import { toggleMute, toggleFullscreen } from './controls.js';
import { openLogModal, closeLogModal } from './logs.js';
import { start } from './webrtc.js';

// Wire up UI event listeners
document.getElementById('overlayMuteBtn').addEventListener('click', toggleMute);
document.getElementById('overlayFsBtn').addEventListener('click', toggleFullscreen);
document.getElementById('notify-btn').addEventListener('click', toggleNotifications);
document.getElementById('log-btn').addEventListener('click', openLogModal);
document.getElementById('status-btn').addEventListener('click', openLogModal);
document.getElementById('log-close-btn').addEventListener('click', closeLogModal);
document.getElementById('log-modal').addEventListener('click', closeLogModal);
document.getElementById('log-card').addEventListener('click', e => e.stopPropagation());

// Initialize modules
initWaveform();
initStatus();
initNotifications();
initDetection();

// not just init, as this is re-invoked on each reconnect
start().catch(err => { 
  document.body.insertAdjacentHTML('beforeend', `<p>Error: ${err.message}</p>`);
});
