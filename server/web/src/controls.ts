const video = document.getElementById('video') as HTMLVideoElement;

export function toggleMute() {
  video.muted = !video.muted;
  const icon = video.muted ? 'volume_off' : 'volume_up';
  const label = video.muted ? 'Unmute' : 'Mute';
  const btn = document.getElementById('overlayMuteBtn') as HTMLElement;
  btn.innerHTML = `<span class="material-icons">${icon}</span>`;
  btn.setAttribute('aria-label', label);
}

export function toggleFullscreen() {
  if (document.fullscreenElement || document.webkitFullscreenElement) {
    document.exitFullscreen?.();
    document.webkitExitFullscreen?.();
  } else {
    const el = document.getElementById('container') as HTMLElement;
    el?.requestFullscreen?.() ?? el?.webkitRequestFullscreen?.();
  }
}

function updateFsBtn() {
  const inFs = !!(document.fullscreenElement || document.webkitFullscreenElement);
  const btn = document.getElementById('overlayFsBtn') as HTMLElement;
  btn.innerHTML = `<span class="material-icons">${inFs ? 'fullscreen_exit' : 'fullscreen'}</span>`;
  btn.setAttribute('aria-label', inFs ? 'Exit fullscreen' : 'Fullscreen');
}

document.addEventListener('fullscreenchange', updateFsBtn);
document.addEventListener('webkitfullscreenchange', updateFsBtn);
