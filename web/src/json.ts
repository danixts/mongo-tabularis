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

export interface FieldEntry {
  path: string;
  type: string;
}

export function bsonLabel(value: unknown): string {
  const resolved = asJson(value) ?? value;
  if (Array.isArray(resolved)) return "Array";
  if (resolved === null) return "Null";
  if (resolved && typeof resolved === "object") return "Object";
  if (typeof value === "boolean") return "Boolean";
  if (typeof value === "number") return Number.isInteger(value) ? "Int" : "Double";
  if (typeof value === "string") return /^[a-f0-9]{24}$/i.test(value) ? "ObjectId" : "String";
  return "Mixed";
}

export function fieldEntries(document: Record<string, unknown>, depth = 2): FieldEntry[] {
  const entries: FieldEntry[] = [];
  const seen = new Set<string>();

  const walk = (value: unknown, prefix: string, level: number) => {
    const resolved = asJson(value) ?? value;
    if (level > 0 && resolved && typeof resolved === "object" && !Array.isArray(resolved)) {
      for (const [key, child] of Object.entries(resolved as Record<string, unknown>)) {
        const path = prefix ? `${prefix}.${key}` : key;
        if (!seen.has(path)) {
          seen.add(path);
          entries.push({ path, type: bsonLabel(child) });
        }
        walk(child, path, level - 1);
      }
    }
  };

  walk(document, "", depth);
  return entries;
}
