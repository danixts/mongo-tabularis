import { useMemo } from "react";
import { usePluginTheme, usePluginToast } from "@tabularis/plugin-api";
import { pretty } from "./json";
import { highlightJson, type JsonPalette } from "./highlight";

interface JsonViewProps {
  value: unknown;
  maxHeight?: number;
}

export function JsonView({ value, maxHeight = 360 }: JsonViewProps) {
  const { isDark, colors } = usePluginTheme();
  const toast = usePluginToast();
  const code = useMemo(() => pretty(value), [value]);

  const border = colors?.border.subtle ?? "rgba(127,127,127,0.25)";
  const surface = colors?.bg.elevated ?? (isDark ? "#0d1117" : "#ffffff");
  const muted = colors?.text.muted ?? "#8b949e";

  const palette: JsonPalette = {
    key: colors?.text.accent ?? "#79c0ff",
    string: colors?.semantic.string ?? "#a5d6ff",
    number: colors?.semantic.number ?? "#f2cc60",
    boolean: colors?.semantic.boolean ?? "#ff7b72",
    null: colors?.semantic.null ?? muted,
    punctuation: colors?.text.secondary ?? muted,
  };

  const nodes = useMemo(() => highlightJson(code, palette), [code, palette]);

  const copy = () => {
    navigator.clipboard.writeText(code).then(
      () => toast.showInfo("JSON copiado"),
      () => toast.showError("No se pudo copiar"),
    );
  };

  return (
    <div
      style={{
        border: `1px solid ${border}`,
        borderRadius: 8,
        overflow: "hidden",
        background: surface,
      }}
    >
      <div
        style={{
          alignItems: "center",
          color: muted,
          display: "flex",
          justifyContent: "space-between",
          padding: "6px 10px",
          borderBottom: `1px solid ${border}`,
          fontSize: 11,
        }}
      >
        <span style={{ fontFamily: "ui-monospace, monospace", letterSpacing: 0.4 }}>JSON</span>
        <button
          onClick={copy}
          style={{
            background: "transparent",
            border: `1px solid ${border}`,
            borderRadius: 6,
            color: muted,
            cursor: "pointer",
            fontSize: 11,
            padding: "2px 10px",
          }}
        >
          Copiar
        </button>
      </div>
      <pre
        style={{
          margin: 0,
          maxHeight,
          overflow: "auto",
          padding: "10px 12px",
          fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
          fontSize: 12.5,
          lineHeight: 1.55,
          whiteSpace: "pre",
          tabSize: 2,
        }}
      >
        <code>{nodes}</code>
      </pre>
    </div>
  );
}
