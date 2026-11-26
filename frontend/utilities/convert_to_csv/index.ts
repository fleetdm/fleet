const defaultFieldSortFunc = (fields: string[]) => fields;

interface ConvertToCSV {
  objArray: any[]; // TODO: typing
  fieldSortFunc?: (fields: string[]) => string[];
  tableHeaders?: any[]; // TODO: typing
}

// Mirrors sanitizeCSVFormula in server/service/hosts.go so client and server exports neutralize the same values.
const sanitizeCSVFormula = (value: string): string => {
  // Spreadsheets may trim a cell before parsing it, so the formula character is not necessarily the first byte.
  const trimmed = value.trim();
  if (!trimmed || !"=+-@".includes(trimmed[0])) {
    return value;
  }
  // Signed numbers are not formulas; keep numeric columns machine-readable.
  if (Number.isFinite(Number(trimmed))) {
    return value;
  }
  return `'${value}`;
};

const formatFieldForCSV = (value: any): string => {
  // If the value is an object, stringify it first
  if (typeof value === "object") {
    value = JSON.stringify(value);
  }

  // Treat values with leading zeros as strings so csv file doesn't trim leading zeros
  if (/^0\d+$/.test(value)) {
    return `"=""${value}"""`;
  }

  // Escape double quotes in the value by doubling them
  if (typeof value === "string") {
    value = sanitizeCSVFormula(value).replace(/"/g, '""');
  }

  // Wrap the value in double quotes to enclose any value that may
  // have a, or a " in it to distinguish them from a comma-separated delimiter
  return `"${value}"`;
};

const convertToCSV = ({
  objArray,
  fieldSortFunc = defaultFieldSortFunc,
  tableHeaders,
}: ConvertToCSV) => {
  const tableHeadersStrings: string[] = tableHeaders
    ? tableHeaders.map((header: { id: string }) => header.id)
    : Object.keys(objArray[0]);

  let fields = fieldSortFunc(tableHeadersStrings);

  // TODO: Revisit after v5 if column names are modified/removed from API response.
  fields = fields.filter((field) => field !== "Host");
  // Revisit end

  const headerRow = fields.map((field) => formatFieldForCSV(field)).join(",");
  const dataRows = objArray.map((row) =>
    fields.map((field) => formatFieldForCSV(row[field])).join(",")
  );

  return [headerRow, ...dataRows].join("\n");
};

export default convertToCSV;
