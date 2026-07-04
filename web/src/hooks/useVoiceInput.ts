import { useCallback, useMemo, useState } from 'react';

/**
 * Voice input state machine (CAL-163). It is the browser counterpart to the
 * server-side voice gateway: voice is attempted, and any device problem —
 * unsupported browser, denied permission, capture error — degrades to typed
 * input so the candidate is never trapped. Voice is off by default (a flag) in
 * the text-only POC; the live audio pipeline to the STT provider is post-win.
 */
export type VoiceState =
  | 'unsupported' // the browser has no getUserMedia
  | 'idle' // ready, not capturing
  | 'requesting' // asking for mic permission
  | 'recording' // capturing
  | 'denied' // permission refused
  | 'error'; // device/capture failure

export interface VoiceInput {
  supported: boolean;
  state: VoiceState;
  error: string | null;
  /** true when the caller should fall back to the text field. */
  degraded: boolean;
  start: () => Promise<boolean>;
  stop: () => void;
}

function hasGetUserMedia(): boolean {
  return typeof navigator !== 'undefined' && typeof navigator.mediaDevices?.getUserMedia === 'function';
}

export function useVoiceInput(): VoiceInput {
  const supported = useMemo(() => hasGetUserMedia(), []);
  const [state, setState] = useState<VoiceState>(supported ? 'idle' : 'unsupported');
  const [error, setError] = useState<string | null>(null);

  const start = useCallback(async (): Promise<boolean> => {
    if (!supported) {
      setState('unsupported');
      return false;
    }
    setState('requesting');
    setError(null);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      // POC: we only prove device access + permission here; there is no capture
      // pipeline yet, so release the mic immediately rather than hold it open.
      stream.getTracks().forEach((track) => track.stop());
      setState('recording');
      return true;
    } catch (err) {
      const name = err instanceof DOMException ? err.name : '';
      const denied = name === 'NotAllowedError' || name === 'SecurityError' || name === 'PermissionDeniedError';
      setState(denied ? 'denied' : 'error');
      setError(denied ? 'Microphone permission was denied — you can type your answer instead.' : 'Could not access the microphone — you can type your answer instead.');
      return false;
    }
  }, [supported]);

  const stop = useCallback(() => {
    setState((s) => (s === 'recording' || s === 'requesting' ? 'idle' : s));
  }, []);

  const degraded = state === 'unsupported' || state === 'denied' || state === 'error';

  return { supported, state, error, degraded, start, stop };
}
