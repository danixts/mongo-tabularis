import { useEffect, useMemo, useRef, useState } from "react";
import {
  defineSlot,
  usePluginModal,
  usePluginQuery,
  usePluginTheme,
} from "@tabularis/plugin-api";
import { JsonView } from "./JsonView";
import { asJson, fieldEntries, type FieldEntry } from "./json";
import { operators } from "./operators";

interface Suggestion {
  display: string;
  hint: string;
  insert: string;
}

function rowsToDocuments(columns: string[], rows: unknown[][]) {
  return rows.map((row) => {
    const document: Record<string, unknown> = {};
    columns.forEach((column, index) => {
      document[column] = asJson(row[index]) ?? row[index];
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

  const [filter, setFilter] = useState("{}");
  const [fields, setFields] = useState<FieldEntry[]>([]);
  const [documents, setDocuments] = useState<Record<string, unknown>[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [fieldsError, setFieldsError] = useState<string | null>(null);
  const [caret, setCaret] = useState(0);

  const border = colors?.border.default ?? "rgba(127,127,127,0.3)";
  const subtle = colors?.border.subtle ?? "rgba(127,127,127,0.18)";
  const inputBg = colors?.bg.input ?? (isDark ? "#0d1117" : "#fff");
  const panelBg = colors?.bg.elevated ?? (isDark ? "#161b22" : "#f6f8fa");
  const text = colors?.text.primary ?? (isDark ? "#e6edf3" : "#1f2328");
  const muted = colors?.text.muted ?? "#8b949e";
  const accent = colors?.accent.primary ?? "#2563eb";
  const hover = colors?.surface.hover ?? "rgba(127,127,127,0.12)";
  const typeColor = colors?.text.accent ?? accent;

  const token = useMemo(() => tokenAt(filter, caret), [filter, caret]);

  const suggestions = useMemo<Suggestion[]>(() => {
    const needle = token.value.toLowerCase();
    if (needle.startsWith("$")) {
      return operators
        .filter((entry) => entry.op.toLowerCase().includes(needle))
        .map((entry) => ({ display: entry.op, hint: entry.desc, insert: `${entry.op}: ` }));
    }
    const matches = needle
      ? fields.filter((entry) => entry.path.toLowerCase().includes(needle))
      : fields;
    return matches
      .slice(0, 50)
      .map((entry) => ({ display: entry.path, hint: entry.type, insert: `"${entry.path}"` }));
  }, [fields, token]);

  const loadFields = async () => {
    setFieldsError(null);
    try {
      const result = await executeQuery(`db.${table}.aggregate([{ "$limit": 1 }])`);
      const sample = rowsToDocuments(result.columns, result.rows.slice(0, 1))[0] ?? {};
      const entries = fieldEntries(sample);
      setFields(
        entries.length > 0 ? entries : result.columns.map((column) => ({ path: column, type: "" })),
      );
    } catch (err) {
      setFieldsError(err instanceof Error ? err.message : String(err));
    }
  };

  const run = async (expression: string) => {
    setError(null);
    const trimmed = expression.trim();
    const where = trimmed && trimmed !== "{}" ? ` WHERE ${trimmed}` : "";
    try {
      const result = await executeQuery(`SELECT * FROM "${table}"${where}`);
      setDocuments(rowsToDocuments(result.columns, result.rows));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setDocuments([]);
    }
  };

  useEffect(() => {
    void loadFields();
    void run("{}");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const insert = (suggestion: Suggestion) => {
    const before = filter.slice(0, token.start);
    const after = filter.slice(caret);
    const next = before + suggestion.insert + after;
    const nextCaret = token.start + suggestion.insert.length;
    setFilter(next);
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
    if (event.key === "Tab" && suggestions.length > 0 && token.value) {
      event.preventDefault();
      insert(suggestions[0]);
      return;
    }
    if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
      event.preventDefault();
      void run(filter);
    }
  };

  const syncCaret = () => {
    const node = textareaRef.current;
    if (node) setCaret(node.selectionStart);
  };

  const labelStyle: React.CSSProperties = {
    color: muted,
    fontSize: 11,
    fontWeight: 600,
    letterSpacing: 0.4,
    textTransform: "uppercase",
  };

  const usingOperators = token.value.startsWith("$");

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        gap: 16,
        color: text,
        padding: 20,
        width: 620,
        maxWidth: "100%",
        boxSizing: "border-box",
      }}
    >
      <div style={{ fontSize: 13, color: muted, lineHeight: 1.5 }}>
        Filtro sobre <strong style={{ color: text }}>{table}</strong>. Sintaxis MongoDB, por ejemplo{" "}
        <code style={{ color: typeColor }}>{`{ "body.total": { "$gt": 0 } }`}</code>.
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        <div style={labelStyle}>Consulta</div>
        <textarea
          ref={textareaRef}
          value={filter}
          spellCheck={false}
          rows={3}
          placeholder="{}"
          onChange={(event) => {
            setFilter(event.target.value);
            setCaret(event.target.selectionStart);
          }}
          onKeyUp={syncCaret}
          onClick={syncCaret}
          onKeyDown={onKeyDown}
          style={{
            background: inputBg,
            border: `1px solid ${border}`,
            borderRadius: 8,
            color: text,
            fontFamily: "ui-monospace, SFMono-Regular, monospace",
            fontSize: 14,
            lineHeight: 1.5,
            padding: "10px 12px",
            resize: "vertical",
            width: "100%",
            boxSizing: "border-box",
          }}
        />
        <div style={{ color: muted, fontSize: 11 }}>
          Escribe un campo o <code style={{ color: typeColor }}>$</code> para sugerencias · Tab inserta · Ctrl/Cmd+Enter aplica
        </div>
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        <div style={{ alignItems: "center", display: "flex", justifyContent: "space-between" }}>
          <div style={labelStyle}>
            {usingOperators ? "Operadores" : `Campos (${fields.length})`}
          </div>
          {!usingOperators && (
            <button
              onClick={() => void loadFields()}
              style={{
                background: "transparent",
                border: `1px solid ${subtle}`,
                borderRadius: 6,
                color: muted,
                cursor: "pointer",
                fontSize: 11,
                padding: "2px 8px",
              }}
            >
              Recargar
            </button>
          )}
        </div>
        <div
          style={{
            background: panelBg,
            border: `1px solid ${subtle}`,
            borderRadius: 8,
            maxHeight: 160,
            overflow: "auto",
            padding: 4,
          }}
        >
          {suggestions.length === 0 ? (
            <div style={{ color: muted, fontSize: 12, padding: "8px 10px" }}>
              {fieldsError ? `Error cargando campos: ${fieldsError}` : "Sin coincidencias"}
            </div>
          ) : (
            suggestions.map((suggestion) => (
              <button
                key={suggestion.display}
                onClick={() => insert(suggestion)}
                onMouseEnter={(e) => (e.currentTarget.style.background = hover)}
                onMouseLeave={(e) => (e.currentTarget.style.background = "transparent")}
                style={{
                  alignItems: "center",
                  background: "transparent",
                  border: "none",
                  borderRadius: 6,
                  color: text,
                  cursor: "pointer",
                  display: "flex",
                  fontFamily: "ui-monospace, SFMono-Regular, monospace",
                  fontSize: 12.5,
                  justifyContent: "space-between",
                  gap: 12,
                  padding: "6px 10px",
                  textAlign: "left",
                  width: "100%",
                }}
              >
                <span>{suggestion.display}</span>
                <span style={{ color: typeColor, fontSize: 11 }}>{suggestion.hint}</span>
              </button>
            ))
          )}
        </div>
      </div>

      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <button
          onClick={() => void run(filter)}
          disabled={loading}
          style={{
            background: accent,
            border: "none",
            borderRadius: 8,
            color: "#fff",
            cursor: loading ? "default" : "pointer",
            fontSize: 14,
            fontWeight: 600,
            opacity: loading ? 0.6 : 1,
            padding: "8px 18px",
          }}
        >
          {loading ? "Buscando" : "Aplicar filtro"}
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
            padding: "10px 12px",
            whiteSpace: "pre-wrap",
          }}
        >
          {error}
        </div>
      )}

      {documents.length > 0 && (
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          <div style={labelStyle}>Resultado</div>
          <JsonView value={documents} maxHeight={400} />
        </div>
      )}
    </div>
  );
}

const Slot = defineSlot("data-grid.toolbar.actions", ({ context }) => {
  const { openModal } = usePluginModal();
  const { colors } = usePluginTheme();
  const table = context.tableName ?? "";

  return (
    <button
      onClick={() =>
        openModal({
          title: `Filtro Mongo · ${table || "colección"}`,
          size: "xl",
          content: <FilterPanel table={table} />,
        })
      }
      title="Filtro MongoDB"
      style={{
        background: colors?.accent.primary ?? "#2563eb",
        border: "none",
        borderRadius: 6,
        color: "#fff",
        cursor: "pointer",
        fontSize: 12,
        fontWeight: 600,
        padding: "5px 12px",
      }}
    >
      Filtro Mongo
    </button>
  );
});

const FilterToolbarButton = Slot.component;

(FilterToolbarButton as unknown as { default: unknown }).default = FilterToolbarButton;

export default FilterToolbarButton;
