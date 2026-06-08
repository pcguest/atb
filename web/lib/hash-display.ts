/** Truncate a 64-char SHA-256 hex digest for at-a-glance display. */
export function truncateSha256(hash: string, head = 12, tail = 12): string {
  if (hash.length <= head + tail + 1) {
    return hash;
  }
  return `${hash.slice(0, head)}…${hash.slice(-tail)}`;
}

export async function copyTextToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}
