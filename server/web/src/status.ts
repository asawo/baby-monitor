import type { ServiceStatus } from './api_types.js';

export const StreamStatus = Object.freeze({
  UNKNOWN: 'unknown',
  CONNECTED: 'connected',
  RECONNECTING: 'reconnecting',
  FAILED: 'failed',
} as const);

export type StreamStatusValue = typeof StreamStatus[keyof typeof StreamStatus];

let streamStatus: StreamStatusValue = StreamStatus.UNKNOWN;
const statusBtn = document.getElementById('status-btn') as HTMLElement;

export function getStreamStatus(): StreamStatusValue { return streamStatus; }

export function updateStatusLabel(text: string) {
  (document.getElementById('status-label') as HTMLElement).textContent = text;
}

export function setStreamStatus(status: StreamStatusValue) {
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
    case StreamStatus.CONNECTED:
      statusBtn.className = 'ok';
      updateStatusLabel('Connected');
      break;
  }
}

export async function pollStatus() {
  // Don't overwrite a stream-level error with a server-health indicator
  if (streamStatus === StreamStatus.RECONNECTING || streamStatus === StreamStatus.FAILED) return;
  try {
    const res = await fetch('/api/status');
    const services: ServiceStatus[] = await res.json();
    const down = services.filter(s => !s.active);
    statusBtn.className = down.length === 0 ? 'ok' : 'err';
    updateStatusLabel(down.length === 0 ? 'Connected' : 'Issue');
  } catch {
    statusBtn.className = 'err';
    updateStatusLabel('Issue');
  }
}

export function init() {
  pollStatus();
  setInterval(pollStatus, 10_000);
}
