import { defineSlot, usePluginModal, usePluginTheme } from "@tabularis/plugin-api";
import { JsonView } from "./JsonView";
import { asJson } from "./json";

const Slot = defineSlot("data-grid.context-menu.items", ({ context }) => {
  const { openModal } = usePluginModal();
  const { colors } = usePluginTheme();

  const column = context.columnName;
  const value = column ? context.rowData?.[column] : context.rowData;
  const display = asJson(value) ?? value;
  const title = column ? `Campo · ${column}` : "Documento";

  return (
    <button
      onClick={() =>
        openModal({
          title,
          size: "lg",
          content: <JsonView value={display} maxHeight={520} />,
        })
      }
      style={{
        background: "transparent",
        border: "none",
        color: colors?.text.primary ?? "inherit",
        cursor: "pointer",
        display: "block",
        fontSize: 13,
        padding: "6px 12px",
        textAlign: "left",
        width: "100%",
      }}
    >
      Ver JSON formateado
    </button>
  );
});

const CellJsonItem = Slot.component;

(CellJsonItem as unknown as { default: unknown }).default = CellJsonItem;

export default CellJsonItem;
