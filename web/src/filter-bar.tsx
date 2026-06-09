import { useEffect, useMemo, useRef, useState } from "react";
import {
  defineSlot,
  usePluginModal,
  usePluginQuery,
  usePluginTheme,
} from "@tabularis/plugin-api";
import { JsonView } from "./JsonView";
import { asJson, fieldPaths } from "./json";

function rowsToDocuments(columns: string[], rows: unknown[][]) {
  return rows.map((row) => {
    const document: Record<string, unknown> = {};
    columns.forEach((column, index) => {
      const value = row[index];
      document[column] = asJson(value) ?? value;
    });
    return document;
  });
}

const IDENTIFIER = /[A-Za-z0-9_$.]/;

function tokenAt(text: string, caret: number) {
  let start = caret;
  while (start > 0 && IDENTIFIER.test(text[start - 1])) {
    start -= 1;
  }
  return { start, value: text.slice(start, caret) };
}

interface PanelProps {
  table: string;
}

function FilterPanel({ table }: PanelProps) {
  const { executeQuery, loading } = usePluginQuery();
  const { colors, isDark } = usePluginTheme();
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const [filter, setFilter] = useState("{ }");
  const [fields, setFields] = useState<string[]>([]);
  const [documents, setDocuments] = useState<Record<string, unknown>[]>([]);
  const [error, setError] = useState<string | null>(null);

  const [caret, setCaret] = useState(0);
  const [suggestOpen, setSuggestOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);

  const border = colors?.border.default ?? "rgba(127,127,127,0.3)";
  const inputBg = colors?.bg.input ?? (isDark ? "#0d1117" : "#fff");
  const overlayBg = colors?.bg.overlay ?? (isDark ? "#161b22" : "#fff");
  const text = colors?.text.primary ?? (isDark ? "#e6edf3" : "#1f2328");
  const muted = colors?.text.muted ?? "#8b949e";
  const accent = colors?.accent.primary ?? "#3b82f6";

  const token = useMemo(() => tokenAt(filter, caret), [filter, caret]);
  const suggestions = useMemo(() => {
    const needle = token.value.toLowerCase();
    if (needle.length === 0) return [];
    return fields
      .filter((path) => path.toLowerCase().includes(needle) && path !== token.value)
      .slice(0, 8);
  }, [fields, token]);

  useEffect(() => {
    setActiveIndex(0);
  }, [token.value]);

  const run = async (expression: string) => {
    setError(null);
    const trimmed = expression.trim();
    const where = trimmed && trimmed !== "{}" && trimmed !== "{ }" ? ` WHERE ${trimmed}` : "";
    try {
      const result = await executeQuery(`SELECT * FROM "${table}"${where}`);
      setDocuments(rowsToDocuments(result.columns, result.rows));
      if (fields.length === 0 && result.columns.length > 0) {
        const sample = rowsToDocuments(result.columns, result.rows.slice(0, 1))[0] ?? {};
        setFields(fieldPaths(sample));
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setDocuments([]);
    }
  };

  useEffect(() => {
    void run("{}");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const applySuggestion = (path: string) => {
    const before = filter.slice(0, token.start);
    const after = filter.slice(caret);
    const insertion = `"${path}"`;
    const next = before + insertion + after;
    const nextCaret = token.start + insertion.length;
    setFilter(next);
    setSuggestOpen(false);
    requestAnimationFrame(() => {
      const node = textareaRef.current;
      if (node) {
        node.focus();
        node.setSelectionRange(nextCaret, nextCaret);
        setCaret(nextCaret);
      }
    });
  };

  const onKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (suggestOpen && suggestions.length > 0) {
      if (event.key === "ArrowDown") {
        event.preventDefault();
        setActiveIndex((index) => (index + 1) % suggestions.length);
        return;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        setActiveIndex((index) => (index - 1 + suggestions.length) % suggestions.length);
        return;
      }
      if (event.key === "Enter" || event.key === "Tab") {
        event.preventDefault();
        applySuggestion(suggestions[activeIndex]);
        return;
      }
      if (event.key === "Escape") {
        event.preventDefault();
        setSuggestOpen(false);
        return;
      }
    }
    if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
      event.preventDefault();
      void run(filter);
    }
  };

  const syncCaret = () => {
    const node = textareaRef.current;
    if (node) {
      setCaret(node.selectionStart);
      setSuggestOpen(true);
    }
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 10, color: text, minWidth: 540 }}>
      <div style={{ fontSize: 12, color: muted }}>
        Filtro MongoDB sobre <strong>{table}</strong> — sintaxis Compass. Escribe un campo para
        autocompletar; <kbd>↑↓</kbd> navega, <kbd>Tab</kbd> inserta, <kbd>⌘/Ctrl+Enter</kbd> aplica.
      </div>

      <div style={{ position: "relative" }}>
        <textarea
          ref={textareaRef}
          value={filter}
          spellCheck={false}
          rows={3}
          onChange={(event) => {
            setFilter(event.target.value);
            setCaret(event.target.selectionStart);
            setSuggestOpen(true);
          }}
          onKeyUp={syncCaret}
          onClick={syncCaret}
          onKeyDown={onKeyDown}
          onBlur={() => window.setTimeout(() => setSuggestOpen(false), 120)}
          style={{
            background: inputBg,
            border: `1px solid ${border}`,
            borderRadius: 8,
            color: text,
            fontFamily: "monospace",
            fontSize: 13,
            padding: "8px 10px",
            resize: "vertical",
            width: "100%",
            boxSizing: "border-box",
          }}
        />

        {suggestOpen && suggestions.length > 0 && (
          <ul
            style={{
              position: "absolute",
              zIndex: 20,
              left: 8,
              right: 8,
              top: "100%",
              margin: "4px 0 0",
              padding: 4,
              listStyle: "none",
              background: overlayBg,
              border: `1px solid ${border}`,
              borderRadius: 8,
              boxShadow: "0 8px 24px rgba(0,0,0,0.25)",
              maxHeight: 220,
              overflow: "auto",
            }}
          >
            {suggestions.map((path, index) => (
              <li key={path}>
                <button
                  onMouseDown={(event) => {
                    event.preventDefault();
                    applySuggestion(path);
                  }}
                  onMouseEnter={() => setActiveIndex(index)}
                  style={{
                    background: index === activeIndex ? colors?.surface.hover ?? "rgba(127,127,127,0.15)" : "transparent",
                    border: "none",
                    borderRadius: 6,
                    color: text,
                    cursor: "pointer",
                    display: "block",
                    fontFamily: "monospace",
                    fontSize: 12,
                    padding: "5px 8px",
                    textAlign: "left",
                    width: "100%",
                  }}
                >
                  {path}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <button
          onClick={() => void run(filter)}
          disabled={loading}
          style={{
            background: accent,
            border: "none",
            borderRadius: 8,
            color: "#fff",
            cursor: loading ? "default" : "pointer",
            fontSize: 13,
            fontWeight: 600,
            opacity: loading ? 0.6 : 1,
            padding: "6px 16px",
          }}
        >
          {loading ? "Buscando…" : "Aplicar filtro"}
        </button>
        <span style={{ color: muted, fontSize: 12 }}>{documents.length} documentos</span>
      </div>

      {error && (
        <div
          style={{
            background: "rgba(248,81,73,0.12)",
            border: `1px solid ${colors?.accent.error ?? "#f85149"}`,
            borderRadius: 8,
            color: colors?.accent.error ?? "#f85149",
            fontSize: 12,
            padding: "8px 10px",
            whiteSpace: "pre-wrap",
          }}
        >
          {error}
        </div>
      )}

      {documents.length > 0 && <JsonView value={documents} maxHeight={420} />}
    </div>
  );
}

const Slot = defineSlot("data-grid.toolbar.actions", ({ context }) => {
  const { openModal } = usePluginModal();
  const { colors } = usePluginTheme();

  if (!context.tableName) {
    return null;
  }

  return (
    <button
      onClick={() =>
        openModal({
          title: `Filtro Mongo · ${context.tableName}`,
          size: "lg",
          content: <FilterPanel table={context.tableName} />,
        })
      }
      title="Filtro estilo Compass"
      style={{
        background: "transparent",
        border: `1px solid ${colors?.border.default ?? "rgba(127,127,127,0.3)"}`,
        borderRadius: 6,
        color: colors?.text.secondary ?? "inherit",
        cursor: "pointer",
        fontSize: 12,
        padding: "4px 10px",
      }}
    >
      ⛃ Filtro
    </button>
  );
});

export default Slot.component;
