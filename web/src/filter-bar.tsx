import { useEffect, useState } from "react";
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

function quoteIdentifier(name: string) {
  return `"${name}"`;
}

interface PanelProps {
  table: string;
}

function FilterPanel({ table }: PanelProps) {
  const { executeQuery, loading } = usePluginQuery();
  const { colors, isDark } = usePluginTheme();
  const [filter, setFilter] = useState("{ }");
  const [fields, setFields] = useState<string[]>([]);
  const [documents, setDocuments] = useState<Record<string, unknown>[]>([]);
  const [error, setError] = useState<string | null>(null);

  const border = colors?.border.default ?? "rgba(127,127,127,0.3)";
  const inputBg = colors?.bg.input ?? (isDark ? "#0d1117" : "#fff");
  const text = colors?.text.primary ?? (isDark ? "#e6edf3" : "#1f2328");
  const muted = colors?.text.muted ?? "#8b949e";
  const accent = colors?.accent.primary ?? "#3b82f6";

  const run = async (expression: string) => {
    setError(null);
    const trimmed = expression.trim();
    const where = trimmed && trimmed !== "{}" && trimmed !== "{ }" ? ` WHERE ${trimmed}` : "";
    try {
      const result = await executeQuery(`SELECT * FROM ${quoteIdentifier(table)}${where}`);
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

  const insertField = (path: string) => {
    setFilter((current) => {
      const body = current.replace(/^\s*\{|\}\s*$/g, "").trim();
      const next = body ? `${body}, "${path}": ` : `"${path}": `;
      return `{ ${next}}`;
    });
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 10, color: text, minWidth: 520 }}>
      <div style={{ fontSize: 12, color: muted }}>
        Filtro MongoDB sobre <strong>{table}</strong> — sintaxis Compass, p. ej.{" "}
        <code>{`{ "body.total": 96 }`}</code> o <code>{`{ durationMs: { $gt: 0 } }`}</code>
      </div>

      {fields.length > 0 && (
        <div style={{ display: "flex", flexWrap: "wrap", gap: 4, maxHeight: 88, overflow: "auto" }}>
          {fields.map((path) => (
            <button
              key={path}
              onClick={() => insertField(path)}
              title="Insertar campo"
              style={{
                background: "transparent",
                border: `1px solid ${border}`,
                borderRadius: 999,
                color: muted,
                cursor: "pointer",
                fontFamily: "monospace",
                fontSize: 11,
                padding: "2px 8px",
              }}
            >
              {path}
            </button>
          ))}
        </div>
      )}

      <textarea
        value={filter}
        onChange={(event) => setFilter(event.target.value)}
        spellCheck={false}
        rows={3}
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
