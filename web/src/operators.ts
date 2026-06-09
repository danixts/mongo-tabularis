export interface OperatorEntry {
  op: string;
  desc: string;
}

export const operators: OperatorEntry[] = [
  { op: "$eq", desc: "igual a" },
  { op: "$ne", desc: "distinto de" },
  { op: "$gt", desc: "mayor que" },
  { op: "$gte", desc: "mayor o igual" },
  { op: "$lt", desc: "menor que" },
  { op: "$lte", desc: "menor o igual" },
  { op: "$in", desc: "en el array" },
  { op: "$nin", desc: "no en el array" },
  { op: "$exists", desc: "campo existe" },
  { op: "$regex", desc: "expresión regular" },
  { op: "$type", desc: "tipo BSON" },
  { op: "$size", desc: "tamaño de array" },
  { op: "$all", desc: "contiene todos" },
  { op: "$elemMatch", desc: "elemento coincide" },
  { op: "$not", desc: "negación" },
  { op: "$and", desc: "y lógico" },
  { op: "$or", desc: "o lógico" },
  { op: "$nor", desc: "ni lógico" },
];
