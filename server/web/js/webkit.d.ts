// Ambient augmentations for non-standard WebKit fullscreen APIs

interface Document {
  webkitFullscreenElement: Element | null;
  webkitExitFullscreen(): Promise<void>;
}

interface HTMLElement {
  webkitRequestFullscreen(): Promise<void>;
}
