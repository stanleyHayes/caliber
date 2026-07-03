import { describe, expect, it } from 'vitest';

import { serializeJsonLd } from './jsonld';

// U+2028 / U+2029 are line terminators — build them from code points, never as
// raw literals, so this test file itself stays parseable.
const LS = String.fromCharCode(0x2028);
const PS = String.fromCharCode(0x2029);

describe('serializeJsonLd', () => {
  it('escapes a </script> breakout while round-tripping as JSON', () => {
    const data = { name: 'Acme </script><script>alert(1)</script>' };
    const out = serializeJsonLd(data);

    // The literal breakout sequence must not survive into the <script> body.
    expect(out).not.toContain('</script>');
    expect(out).not.toContain('<');
    expect(out).toContain('\\u003c');
    // ...but it is still valid, lossless JSON.
    expect(JSON.parse(out)).toEqual(data);
  });

  it('escapes ampersands and the U+2028/U+2029 line separators', () => {
    const data = { a: 'x & y', b: LS + PS };
    const out = serializeJsonLd(data);

    expect(out).toContain('\\u0026');
    expect(out).toContain('\\u2028');
    expect(out).toContain('\\u2029');
    expect(out).not.toContain(LS);
    expect(out).not.toContain(PS);
    expect(JSON.parse(out)).toEqual(data);
  });

  it('leaves ordinary structured data intact', () => {
    const data = { '@type': 'Organization', name: 'Project Caliber' };
    expect(JSON.parse(serializeJsonLd(data))).toEqual(data);
  });
});
