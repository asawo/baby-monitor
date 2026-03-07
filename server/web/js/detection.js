const cryAlert = document.getElementById('cry-alert');
const detectErrorEl = document.getElementById('detect-error');
const CRY_RECENT_SECONDS = 10;

async function pollCry() {
  try {
    const res = await fetch('/api/cry');
    const data = await res.json();
    const recent = data.detected_at !== null && data.seconds_ago < CRY_RECENT_SECONDS;
    cryAlert.classList.toggle('visible', recent);
  } catch {}
}

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

export function init() {
  pollCry();
  pollDetectStatus();
  setInterval(pollCry, 5000);
  setInterval(pollDetectStatus, 10_000);
}
