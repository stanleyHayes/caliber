// downloadTextFile triggers a browser download of text content as a named file.
// Guarded so it is a no-op in non-browser environments (e.g. tests/SSR).
export function downloadTextFile(filename: string, content: string, type = 'application/json'): void {
  if (typeof document === 'undefined' || typeof URL.createObjectURL !== 'function') {
    return;
  }
  const url = URL.createObjectURL(new Blob([content], { type }));
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
