const defaultFieldSortFunc = (fields: string[]) => fields;

interface ConvertToCSV {
  objArray: any; // TODO: typing
  fieldSortFunc?: (fields: string[]) => string[];
  tableHeaders?: any[]; // TODO: typing
}

const convertToCSV = ({
  objArray,
  fieldSortFunc = defaultFieldSortFunc,
  tableHeaders,
}: ConvertToCSV) => {
  const tableHeadersStrings: string[] = tableHeaders
    ? tableHeaders.map((header: { id: string }) => header.id) // TODO: typing
    : Object.keys(objArray[0]);

  const fields = fieldSortFunc(tableHeadersStrings);

  // TODO: Remove after v5 when host_hostname is removed rom API response.
  const hostNameIndex = fields.indexOf("host_hostname");
  if (hostNameIndex >= 0) {
    fields.splice(hostNameIndex, 1);
  }
  // Remove end
  const jsonFields = fields.map((field) => JSON.stringify(field));
  const rows = objArray.map((row: any) => {
    // TODO: typing
    return fields.map((field) => JSON.stringify(row[field])).join(",");
  });

  rows.unshift(jsonFields.join(","));

  return rows.join("\n");
};

export default convertToCSV;
