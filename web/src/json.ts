export function asJson(value: unknown): unknown | null {
  if (value && typeof value === "object") {
    return value;
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
      try {
        const parsed = JSON.parse(trimmed);
        if (parsed && typeof parsed === "object") {
          return parsed;
        }
      } catch {
        return null;
      }
    }
  }
  return null;
}

export function pretty(value: unknown): string {
  if (typeof value === "string") {
    try {
      return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
      return value;
    }
  }
  return JSON.stringify(value, null, 2);
}

export function fieldPaths(document: Record<string, unknown>, depth = 2): string[] {
  const paths: string[] = [];

  const walk = (value: unknown, prefix: string, level: number) => {
    const resolved = asJson(value) ?? value;
    if (level > 0 && resolved && typeof resolved === "object" && !Array.isArray(resolved)) {
      for (const [key, child] of Object.entries(resolved as Record<string, unknown>)) {
        const path = prefix ? `${prefix}.${key}` : key;
        paths.push(path);
        walk(child, path, level - 1);
      }
    }
  };

  walk(document, "", depth);
  return Array.from(new Set(paths));
}
