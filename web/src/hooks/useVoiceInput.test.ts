import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useVoiceInput } from './useVoiceInput';

function setMediaDevices(getUserMedia: unknown) {
  Object.defineProperty(globalThis.navigator, 'mediaDevices', {
    configurable: true,
    value: getUserMedia === undefined ? undefined : { getUserMedia },
  });
}

afterEach(() => {
  setMediaDevices(undefined);
  vi.restoreAllMocks();
});

describe('useVoiceInput', () => {
  it('reports unsupported and degrades when the browser lacks getUserMedia', async () => {
    setMediaDevices(undefined);
    const { result } = renderHook(() => useVoiceInput());
    expect(result.current.supported).toBe(false);
    expect(result.current.state).toBe('unsupported');
    expect(result.current.degraded).toBe(true);
    await act(async () => {
      await result.current.start();
    });
    expect(result.current.state).toBe('unsupported');
  });

  it('records when permission is granted, releasing the mic', async () => {
    const stop = vi.fn();
    const getUserMedia = vi.fn().mockResolvedValue({ getTracks: () => [{ stop }] });
    setMediaDevices(getUserMedia);
    const { result } = renderHook(() => useVoiceInput());
    expect(result.current.supported).toBe(true);
    await act(async () => {
      const ok = await result.current.start();
      expect(ok).toBe(true);
    });
    expect(result.current.state).toBe('recording');
    expect(stop).toHaveBeenCalled(); // mic released in the POC (no capture pipeline)
    act(() => result.current.stop());
    expect(result.current.state).toBe('idle');
  });

  it('degrades to typed input when permission is denied', async () => {
    const getUserMedia = vi.fn().mockRejectedValue(new DOMException('no', 'NotAllowedError'));
    setMediaDevices(getUserMedia);
    const { result } = renderHook(() => useVoiceInput());
    await act(async () => {
      const ok = await result.current.start();
      expect(ok).toBe(false);
    });
    expect(result.current.state).toBe('denied');
    expect(result.current.degraded).toBe(true);
    expect(result.current.error).toMatch(/type your answer/i);
  });

  it('degrades on a generic capture error', async () => {
    const getUserMedia = vi.fn().mockRejectedValue(new DOMException('boom', 'NotReadableError'));
    setMediaDevices(getUserMedia);
    const { result } = renderHook(() => useVoiceInput());
    await act(async () => {
      await result.current.start();
    });
    expect(result.current.state).toBe('error');
    expect(result.current.degraded).toBe(true);
  });
});
