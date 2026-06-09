import { useEffect, useMemo, useState } from "react";
import { usePluginTheme, usePluginToast } from "@tabularis/plugin-api";
import { getHighlighter, themeNameFor } from "./shiki";
import { pretty } from "./json";

interface JsonViewProps {
  value: unknown;
  maxHeight?: number;
}

export function JsonView({ value, maxHeight = 360 }: JsonViewProps) {
  const { isDark, colors } = usePluginTheme();
  const toast = usePluginToast();
  const code = useMemo(() => pretty(value), [value]);
  const [html, setHtml] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    getHighlighter()
      .then((highlighter) => {
        if (!active) return;
        setHtml(
          highlighter.codeToHtml(code, {
            lang: "json",
            theme: themeNameFor(isDark),
          }),
        );
      })
      .catch(() => {
        if (active) setHtml(null);
      });
    return () => {
      active = false;
    };
  }, [code, isDark]);

  const border = colors?.border.subtle ?? "rgba(127,127,127,0.25)";
  const surface = colors?.bg.elevated ?? (isDark ? "#0d1117" : "#ffffff");
  const muted = colors?.text.muted ?? "#8b949e";

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
        fontSize: 12,
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "4px 8px",
          borderBottom: `1px solid ${border}`,
          color: muted,
        }}
      >
        <span style={{ fontFamily: "monospace" }}>JSON</span>
        <button
          onClick={copy}
          style={{
            background: "transparent",
            border: `1px solid ${border}`,
            borderRadius: 6,
            color: muted,
            cursor: "pointer",
            fontSize: 11,
            padding: "2px 8px",
          }}
        >
          Copiar
        </button>
      </div>
      <div
        style={{ maxHeight, overflow: "auto", padding: "8px 10px" }}
        className="tabularis-mongo-json"
      >
        {html ? (
          <div dangerouslySetInnerHTML={{ __html: html }} />
        ) : (
          <pre style={{ margin: 0, whiteSpace: "pre-wrap", fontFamily: "monospace" }}>
            {code}
          </pre>
        )}
      </div>
      <style>{`
        .tabularis-mongo-json pre { margin: 0; background: transparent !important; }
        .tabularis-mongo-json code { font-family: ui-monospace, SFMono-Regular, monospace; }
      `}</style>
    </div>
  );
}
