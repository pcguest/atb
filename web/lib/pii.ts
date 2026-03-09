export function isMaskedValue(value: unknown): boolean {
  return value === "[REDACTED]" || value === "***";
}

export function deepClone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

export function setByPath(root: unknown, path: string, nextValue: unknown): unknown {
  const parts = path.split(".").filter(Boolean);
  if (parts.length === 0) {
    return root;
  }

  const out = deepClone(root);
  let current: unknown = out;

  for (let i = 0; i < parts.length; i++) {
    const part = parts[i];
    const isLast = i === parts.length - 1;

    if (Array.isArray(current)) {
      const idx = Number(part);
      if (!Number.isInteger(idx) || idx < 0 || idx >= current.length) {
        return out;
      }
      if (isLast) {
        current[idx] = nextValue;
        return out;
      }
      current = current[idx];
      continue;
    }

    if (current && typeof current === "object") {
      const obj = current as Record<string, unknown>;
      if (!(part in obj)) {
        return out;
      }
      if (isLast) {
        obj[part] = nextValue;
        return out;
      }
      current = obj[part];
      continue;
    }

    return out;
  }

  return out;
}

export function collectMaskedPaths(value: unknown, prefix = "data"): string[] {
  const paths: string[] = [];

  function walk(current: unknown, path: string): void {
    if (isMaskedValue(current)) {
      paths.push(path);
      return;
    }
    if (Array.isArray(current)) {
      current.forEach((item, idx) => walk(item, `${path}.${idx}`));
      return;
    }
    if (current && typeof current === "object") {
      Object.entries(current as Record<string, unknown>).forEach(([key, item]) => {
        walk(item, `${path}.${key}`);
      });
    }
  }

  walk(value, prefix);
  return paths;
}
