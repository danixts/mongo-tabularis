import type { ReactNode } from "react";

export interface JsonPalette {
  key: string;
  string: string;
  number: string;
  boolean: string;
  null: string;
  punctuation: string;
}

const TOKEN = /("(?:\\.|[^"\\])*")(\s*:)?|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|\b(true|false)\b|\b(null)\b/g;

export function highlightJson(text: string, palette: JsonPalette): ReactNode[] {
  const nodes: ReactNode[] = [];
  let last = 0;
  let key = 0;
  let match: RegExpExecArray | null;
  TOKEN.lastIndex = 0;

  const push = (value: string, color: string) => {
    nodes.push(
      <span key={key++} style={{ color }}>
        {value}
      </span>,
    );
  };

  while ((match = TOKEN.exec(text)) !== null) {
    if (match.index > last) {
      push(text.slice(last, match.index), palette.punctuation);
    }
    const [full, str, colon, num, bool, nul] = match;
    if (str !== undefined) {
      if (colon) {
        push(str, palette.key);
        push(colon, palette.punctuation);
      } else {
        push(str, palette.string);
      }
    } else if (num !== undefined) {
      push(num, palette.number);
    } else if (bool !== undefined) {
      push(bool, palette.boolean);
    } else if (nul !== undefined) {
      push(nul, palette.null);
    }
    last = match.index + full.length;
  }
  if (last < text.length) {
    push(text.slice(last), palette.punctuation);
  }
  return nodes;
}
