import type { SlotComponentProps } from "@tabularis/plugin-api";
import { JsonView } from "./JsonView";
import { asJson } from "./json";

function JsonFieldViewer({ context }: SlotComponentProps) {
  const column = context.columnName;
  if (!column || !context.rowData) {
    return null;
  }
  const json = asJson(context.rowData[column]);
  if (json === null) {
    return null;
  }
  return (
    <div style={{ marginTop: 6 }}>
      <JsonView value={json} />
    </div>
  );
}

(JsonFieldViewer as unknown as { default: unknown }).default = JsonFieldViewer;

export default JsonFieldViewer;
