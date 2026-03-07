import { setupAudio, resetAudio } from './waveform.js';
import { StreamStatus, setStreamStatus, pollStatus } from './status.js';

const video = /** @type {HTMLVideoElement} */ (document.getElementById('video'));
const whepUrl = `http://${location.hostname}:8889/baby/whep`;

/** @type {RTCPeerConnection | null} */
let pc = null;
/** @type {number | undefined} */
let reconnectTimer = undefined;

export function reconnect() {
  if (pc) { pc.close(); pc = null; }
  video.srcObject = null;
  resetAudio();
  setStreamStatus(StreamStatus.RECONNECTING);
  setTimeout(() => start().catch(() => reconnect()), 2000);
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
