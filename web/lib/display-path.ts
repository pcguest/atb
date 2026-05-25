/** Shorten bundle paths for dashboard display (demo-friendly). */
export function displayBundlePath(path: string | undefined | null): string {
  if (!path) return "—";
  const marker = "examples/bundles/";
  const idx = path.indexOf(marker);
  if (idx >= 0) return path.slice(idx);
  const parts = path.split(/[/\\]/).filter(Boolean);
  if (parts.length >= 2) return `${parts[parts.length - 2]}/${parts[parts.length - 1]}`;
  return parts.at(-1) ?? path;
}
