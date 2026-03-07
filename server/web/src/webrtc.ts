import { setupAudio, resetAudio } from './waveform.js';
import { StreamStatus, setStreamStatus, pollStatus } from './status.js';

const video = document.getElementById('video') as HTMLVideoElement;
const whepUrl = `http://${location.hostname}:8889/baby/whep`;

const RECONNECT_DELAY_MS = 2000;      // delay before retrying after a close
const DISCONNECT_TIMEOUT_MS = 4000;   // grace period before treating disconnect as failure

let pc: RTCPeerConnection | null = null;
let reconnectTimer: number | undefined = undefined;

export function reconnect() {
  if (pc) { pc.close(); pc = null; }
  video.srcObject = null;
  resetAudio();
  setStreamStatus(StreamStatus.RECONNECTING);
  setTimeout(() => start().catch(() => reconnect()), RECONNECT_DELAY_MS);
}

export async function start() {
  pc = new RTCPeerConnection();

  pc.ontrack = e => {
    if (e.track.kind === 'video') video.srcObject = e.streams[0];
    if (e.track.kind === 'audio') setupAudio(e.streams[0]);
  };

  pc.onconnectionstatechange = () => {
    if (!pc) return;
    switch (pc.connectionState) {
      case 'connected':
        clearTimeout(reconnectTimer);
        setStreamStatus(StreamStatus.CONNECTED);
        pollStatus();
        { const err = document.getElementById('stream-error'); if (err) err.classList.remove('visible'); }
        break;
      case 'disconnected':
        reconnectTimer = setTimeout(reconnect, DISCONNECT_TIMEOUT_MS);
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
    const err = document.getElementById('stream-error');
    if (err) { err.textContent = `Stream not available (${res.status})`; err.classList.add('visible'); }
    return;
  }

  const answer = await res.text();
  await pc.setRemoteDescription({ type: 'answer', sdp: answer });
}
