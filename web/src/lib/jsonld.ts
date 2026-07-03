// serializeJsonLd stringifies a JSON-LD object for safe embedding inside a
// <script type="application/ld+json"> tag. Plain JSON.stringify does NOT escape
// `<`, so a value containing `</script>` would break out of the script element —
// a classic XSS vector. This escapes the characters that can break out of the
// script/HTML context (`<`, `>`, `&`) and the U+2028/U+2029 line separators
// (valid in JSON, invalid in a JS string literal), so the structured data is
// safe even if a dynamic value is ever added to it (CAL-111, XSS-safe rendering).
//
// The pattern is built with RegExp() so the line-separator code points appear as
// \u escapes, not raw (they are line terminators and cannot sit in a regex literal).
const unsafeForScript = new RegExp('[<>&\\u2028\\u2029]', 'g');

export function serializeJsonLd(data: unknown): string {
  return JSON.stringify(data).replace(
    unsafeForScript,
    (ch) => '\\u' + ch.charCodeAt(0).toString(16).padStart(4, '0'),
  );
}
