import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { downloadTextFile } from './download';

describe('downloadTextFile', () => {
  beforeEach(() => {
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn().mockReturnValue('blob:mock'),
      revokeObjectURL: vi.fn(),
    });
  });
  afterEach(() => vi.unstubAllGlobals());

  it('creates a named anchor, clicks it, and cleans up the object URL', () => {
    const click = vi.fn();
    const created: HTMLAnchorElement[] = [];
    const realCreate = document.createElement.bind(document);
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      const el = realCreate(tag) as HTMLAnchorElement;
      if (tag === 'a') {
        el.click = click;
        created.push(el);
      }
      return el;
    });

    downloadTextFile('my-data.json', '{"a":1}');

    expect(URL.createObjectURL).toHaveBeenCalledTimes(1);
    expect(click).toHaveBeenCalledTimes(1);
    expect(created[0].download).toBe('my-data.json');
    expect(created[0].href).toContain('blob:mock');
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:mock');
    expect(document.querySelector('a[download]')).toBeNull(); // the anchor is removed after use
  });

  it('is a no-op when the object-URL API is unavailable (non-browser)', () => {
    vi.stubGlobal('URL', {}); // no createObjectURL
    expect(() => downloadTextFile('x.json', '{}')).not.toThrow();
  });
});
