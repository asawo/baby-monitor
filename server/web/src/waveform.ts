const canvas = document.getElementById('waveform') as HTMLCanvasElement;
const ctx = canvas.getContext('2d') as CanvasRenderingContext2D;
const audioIcon = document.getElementById('audio-icon') as HTMLElement;

// --- Tuning knobs ---
const SCROLL_MS = 300;        // ms per bar scroll (higher = slower)
const SENSITIVITY = 4;        // visual amplification (higher = more reactive)
const YELLOW_THRESHOLD = 0.1; // fraction of max amplitude where green → yellow
const RED_THRESHOLD = 0.3;    // fraction of max amplitude where yellow → red
const BAR_W = 3;              // bar width in px
const BAR_GAP = 1;            // gap between bars in px
const FFT_SIZE = 256;         // analyser FFT size (must be power of 2)
const COLOR_LOW = '#4ade80';  // green  — below YELLOW_THRESHOLD
const COLOR_MID = '#facc15';  // yellow — between thresholds
const COLOR_HIGH = '#f87171'; // red    — above RED_THRESHOLD
// --------------------

let analyser: AnalyserNode | null = null;
let waveBuffer: Uint8Array<ArrayBuffer> | null = null;
let audioCtx: AudioContext | null = null;
let pendingAudioStream: MediaStream | null = null;
let history: Float32Array | null = null;
let histPos = 0, lastTick = 0;
let wasAudioActive = false;
let lastIconLevel = '';

function calcNumBars() { return Math.floor(canvas.width / (BAR_W + BAR_GAP)); }

function drawWaveform() {
  requestAnimationFrame(drawWaveform);
  if (!analyser || !waveBuffer || !history) return;

  const now = performance.now();
  if (now - lastTick < SCROLL_MS) return;
  lastTick = now;

  analyser.getByteTimeDomainData(waveBuffer);
  let peak = 0;
  for (let i = 0; i < waveBuffer.length; i++) {
    peak = Math.max(peak, Math.abs(waveBuffer[i] - 128));
  }
  const norm = peak / 128;
  history[histPos++ % history.length] = norm;

  const level = norm >= RED_THRESHOLD ? 'high' : norm >= YELLOW_THRESHOLD ? 'mid' : 'low';
  if (level !== lastIconLevel) {
    audioIcon.className = `material-icons level-${level}`;
    lastIconLevel = level;
  }

  const w = canvas.width, h = canvas.height;
  ctx.clearRect(0, 0, w, h);

  const step = BAR_W + BAR_GAP;
  const n = history.length;
  for (let i = 0; i < n; i++) {
    const val = history[(histPos + i) % n];
    const barH = val * h * SENSITIVITY;
    ctx.fillStyle = val < YELLOW_THRESHOLD ? COLOR_LOW : val < RED_THRESHOLD ? COLOR_MID : COLOR_HIGH;
    ctx.fillRect(i * step, h - barH, BAR_W, barH);
  }
}

function setupAnalyser(stream: MediaStream) {
  audioCtx = new AudioContext();
  const source = audioCtx.createMediaStreamSource(stream);
  analyser = audioCtx.createAnalyser();
  analyser.fftSize = FFT_SIZE;
  analyser.smoothingTimeConstant = 0;
  waveBuffer = new Uint8Array(analyser.frequencyBinCount);
  source.connect(analyser);
  canvas.width = canvas.offsetWidth;
  canvas.height = canvas.offsetHeight;
  history = new Float32Array(calcNumBars()).fill(0);
  histPos = 0;
  lastTick = 0;
}

export function init() {
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

  drawWaveform();

  document.addEventListener('click', () => {
    if (pendingAudioStream && !audioCtx) {
      wasAudioActive = true;
      setupAnalyser(pendingAudioStream);
    } else if (audioCtx && audioCtx.state === 'suspended') audioCtx.resume();
  });
}

export function setupAudio(stream: MediaStream) {
  pendingAudioStream = stream;
  if (wasAudioActive) setupAnalyser(stream);
}

export function resetAudio() {
  if (audioCtx) { audioCtx.close(); audioCtx = null; analyser = null; }
  pendingAudioStream = null;
}
