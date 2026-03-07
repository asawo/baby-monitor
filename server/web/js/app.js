const video = document.getElementById('video');
const whepUrl = `http://${location.hostname}:8889/baby/whep`;

const canvas = document.getElementById('waveform');
const ctx = canvas.getContext('2d');
let analyser, waveBuffer, audioCtx, pendingAudioStream;
let history, histPos, lastTick;

// --- Tuning knobs ---
const SCROLL_MS = 300;      // ms per bar scroll (higher = slower)
const SENSITIVITY = 4;      // visual amplification (higher = more reactive)
const YELLOW_THRESHOLD = 0.1; // fraction of max amplitude where green → yellow
const RED_THRESHOLD = 0.3;    // fraction of max amplitude where yellow → red
const BAR_W = 3;              // bar width in px
const BAR_GAP = 1;            // gap between bars in px
// --------------------

function calcNumBars() { return Math.floor(canvas.width / (BAR_W + BAR_GAP)); }

function drawWaveform() {
  requestAnimationFrame(drawWaveform);
  if (!analyser) return;

  const now = performance.now();
  if (now - lastTick < SCROLL_MS) return;
  lastTick = now;

  analyser.getByteTimeDomainData(waveBuffer);
  let peak = 0;
  for (let i = 0; i < waveBuffer.length; i++) {
    peak = Math.max(peak, Math.abs(waveBuffer[i] - 128));
  }
  history[histPos++ % history.length] = peak / 128;

  const w = canvas.width, h = canvas.height;
  ctx.clearRect(0, 0, w, h);

  const step = BAR_W + BAR_GAP;
  const n = history.length;
  const colors = ['#4ade80', '#facc15', '#f87171'];

  for (let i = 0; i < n; i++) {
    const val = history[(histPos + i) % n];
    const barH = val * h * SENSITIVITY;
    const tier = val < YELLOW_THRESHOLD ? 0 : val < RED_THRESHOLD ? 1 : 2;
    ctx.fillStyle = colors[tier];
    ctx.fillRect(i * step, h - barH, BAR_W, barH);
  }
}

function setupAnalyser(stream) {
  audioCtx = new AudioContext();
  const source = audioCtx.createMediaStreamSource(stream);
  analyser = audioCtx.createAnalyser();
  analyser.fftSize = 256;
  analyser.smoothingTimeConstant = 0;
  waveBuffer = new Uint8Array(analyser.frequencyBinCount);
  source.connect(analyser);
  canvas.width = canvas.offsetWidth;
  canvas.height = canvas.offsetHeight;
  history = new Float32Array(calcNumBars()).fill(0);
  histPos = 0;
  lastTick = 0;
}

new ResizeObserver(() => {
  const newW = canvas.offsetWidth;
  const newH = canvas.offsetHeight;
  if (newW === canvas.width && newH === canvas.height) return;
  canvas.width = newW;
  canvas.height = newH;
  if (history) {
    const oldN = history.length;
    const newN = calcNumBars();
    const newHistory = new Float32Array(newN).fill(0);
    if (newN >= oldN) {
      for (let i = 0; i < oldN; i++)
        newHistory[newN - oldN + i] = history[(histPos + i) % oldN];
    } else {
      for (let i = 0; i < newN; i++)
        newHistory[i] = history[(histPos + oldN - newN + i) % oldN];
    }
    history = newHistory;
    histPos = 0;
  }
}).observe(canvas);

drawWaveform(); // start the loop once; guards on analyser being non-null

let wasAudioActive = false;

document.addEventListener('click', () => {
  if (pendingAudioStream && !audioCtx) {
    wasAudioActive = true;
    setupAnalyser(pendingAudioStream);
  } else if (audioCtx && audioCtx.state === 'suspended') audioCtx.resume();
});

const StreamStatus = Object.freeze({ UNKNOWN: 'unknown', CONNECTED: 'connected', RECONNECTING: 'reconnecting', FAILED: 'failed' });

let pc = null;
let reconnectTimer = null;
let streamStatus = StreamStatus.UNKNOWN;

function setStreamStatus(status) {
  streamStatus = status;
  switch (status) {
    case StreamStatus.RECONNECTING:
      statusBtn.className = 'err';
      updateStatusLabel('Reconnecting');
      break;
    case StreamStatus.FAILED:
      statusBtn.className = 'err';
      updateStatusLabel('Failed');
      break;
    // CONNECTED is handled by pollStatus on its next tick
  }
}

function reconnect() {
  if (pc) { pc.close(); pc = null; }
  video.srcObject = null;
  if (audioCtx) { audioCtx.close(); audioCtx = null; analyser = null; pendingAudioStream = null; }
  setStreamStatus(StreamStatus.RECONNECTING);
  setTimeout(() => start().catch(() => reconnect()), 2000);
}

async function start() {
  pc = new RTCPeerConnection();

  pc.ontrack = e => {
    if (e.track.kind === 'video') video.srcObject = e.streams[0];
    if (e.track.kind === 'audio') {
      pendingAudioStream = e.streams[0];
      if (wasAudioActive) setupAnalyser(pendingAudioStream);
    }
  };

  pc.onconnectionstatechange = () => {
    switch (pc.connectionState) {
      case 'connected':
        clearTimeout(reconnectTimer);
        streamStatus = StreamStatus.CONNECTED;
        statusBtn.className = 'ok';
        updateStatusLabel('Connected');
        pollStatus();
        break;
      case 'disconnected':
        reconnectTimer = setTimeout(reconnect, 4000);
        setStreamStatus(StreamStatus.RECONNECTING);
        break;
      case 'failed':
      case 'closed':
        clearTimeout(reconnectTimer);
        setStreamStatus(StreamStatus.FAILED);
        reconnect();
        break;
    }
  };

  pc.addTransceiver('video', { direction: 'recvonly' });
  pc.addTransceiver('audio', { direction: 'recvonly' });

  const offer = await pc.createOffer();
  await pc.setLocalDescription(offer);

  const res = await fetch(whepUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/sdp' },
    body: offer.sdp,
  });

  if (!res.ok) {
    document.body.insertAdjacentHTML('beforeend', `<p>Stream unavailable (${res.status})</p>`);
    return;
  }

  const answer = await res.text();
  await pc.setRemoteDescription({ type: 'answer', sdp: answer });
}

function toggleMute() {
  video.muted = !video.muted;
  const icon = video.muted ? 'volume_off' : 'volume_up';
  const label = video.muted ? 'Unmute' : 'Mute';
  const btn = document.getElementById('overlayMuteBtn');
  btn.innerHTML = `<span class="material-icons">${icon}</span>`;
  btn.setAttribute('aria-label', label);
}

function toggleFullscreen() {
  if (document.fullscreenElement || document.webkitFullscreenElement) {
    if (document.exitFullscreen) document.exitFullscreen();
    else if (document.webkitExitFullscreen) document.webkitExitFullscreen();
  } else {
    const el = document.getElementById('container');
    if (el.requestFullscreen) el.requestFullscreen();
    else if (el.webkitRequestFullscreen) el.webkitRequestFullscreen();
  }
}

document.addEventListener('fullscreenchange', updateFsBtn);
document.addEventListener('webkitfullscreenchange', updateFsBtn);

function updateFsBtn() {
  const inFs = !!(document.fullscreenElement || document.webkitFullscreenElement);
  const btn = document.getElementById('overlayFsBtn');
  btn.innerHTML = `<span class="material-icons">${inFs ? 'fullscreen_exit' : 'fullscreen'}</span>`;
  btn.setAttribute('aria-label', inFs ? 'Exit fullscreen' : 'Fullscreen');
}

start().catch(err => {
  document.body.insertAdjacentHTML('beforeend', `<p>Error: ${err.message}</p>`);
});

// --- Status widget ---
const statusBtn = document.getElementById('status-btn');

function updateStatusLabel(text) {
  document.getElementById('status-label').textContent = text;
}

async function pollStatus() {
  // Don't overwrite a stream-level error with a server-health indicator
  if (streamStatus === StreamStatus.RECONNECTING || streamStatus === StreamStatus.FAILED) return;
  try {
    const res = await fetch('/api/status');
    const services = await res.json();
    const down = services.filter(s => !s.active);
    statusBtn.className = down.length === 0 ? 'ok' : 'err';
    updateStatusLabel(down.length === 0 ? 'Connected' : 'Issue');
  } catch {
    statusBtn.className = 'err';
    updateStatusLabel('Issue');
  }
}

pollStatus();
setInterval(pollStatus, 10_000);

const LOG_SECTION_COLORS = {
  'mediamtx':           '#60a5fa', // blue
  'stream.service':     '#34d399', // green
  'monitor-http.service': '#f472b6', // pink
  'detect.service':     '#fb923c', // orange
};
const LOG_SECTION_COLOR_FALLBACK = '#aaa';

const LOG_MAX_LINES = 30;

async function openLogModal() {
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
      const label = s.name;
      const div = document.createElement('div');
      div.className = 'log-section open';
      div.innerHTML = `
        <div class="log-section-header" style="color:${color}">
          <span>${label}</span>
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

function closeLogModal() {
  document.getElementById('log-modal').classList.remove('open');
}

document.addEventListener('keydown', e => {
  if (e.key === 'Escape') closeLogModal();
});

// --- Notifications toggle ---
const notifyBtn = document.getElementById('notify-btn');

async function loadNotificationsState() {
  try {
    const res = await fetch('/api/notifications');
    const data = await res.json();
    updateNotifyBtn(data.enabled);
  } catch {}
}

async function toggleNotifications() {
  try {
    const res = await fetch('/api/notifications', { method: 'POST' });
    const data = await res.json();
    updateNotifyBtn(data.enabled);
  } catch {}
}

function updateNotifyBtn(enabled) {
  notifyBtn.querySelector('.material-icons').textContent = enabled ? 'notifications' : 'notifications_off';
  notifyBtn.setAttribute('aria-label', enabled ? 'Disable notifications' : 'Enable notifications');
  notifyBtn.classList.toggle('disabled', !enabled);
}

loadNotificationsState();

// --- Cry detection indicator ---
const cryAlert = document.getElementById('cry-alert');
const CRY_RECENT_SECONDS = 10;

async function pollCry() {
  try {
    const res = await fetch('/api/cry');
    const data = await res.json();
    const recent = data.detected_at !== null && data.seconds_ago < CRY_RECENT_SECONDS;
    cryAlert.classList.toggle('visible', recent);
  } catch {}
}

pollCry();
setInterval(pollCry, 5000);

// --- Detector error indicator ---
const detectErrorEl = document.getElementById('detect-error');

async function pollDetectStatus() {
  try {
    const res = await fetch('/api/detect-status');
    const data = await res.json();
    if (data.error) {
      detectErrorEl.textContent = 'Detector error: ' + data.error;
      detectErrorEl.classList.add('visible');
    } else {
      detectErrorEl.classList.remove('visible');
    }
  } catch {}
}

pollDetectStatus();
setInterval(pollDetectStatus, 10_000);
