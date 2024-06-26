const defaultFieldSortFunc = (fields: string[]) => fields;

interface ConvertToCSV {
  objArray: any[]; // TODO: typing
  fieldSortFunc?: (fields: string[]) => string[];
  tableHeaders?: any[]; // TODO: typing
}

const convertToCSV = ({
  objArray,
  fieldSortFunc = defaultFieldSortFunc,
  tableHeaders,
}: ConvertToCSV) => {
  const tableHeadersStrings: string[] = tableHeaders
    ? tableHeaders.map((header: { id: string }) => header.id)
    : Object.keys(objArray[0]);

  const fields = fieldSortFunc(tableHeadersStrings);

  // TODO: Revisit after v5 if column names are modified/removed from API response.
  const hostNameIndex = fields.indexOf("Host");
  if (hostNameIndex >= 0) {
    fields.splice(hostNameIndex, 1);
  }
  // Revisit end

  const jsonFields = fields.map((field) => JSON.stringify(field));
  const rows = objArray.map((row: any) => {
    // TODO: typing
    return fields
      .map((field) => {
        // Check if the value of the field is a string and needs to be quoted
        let value = row[field];

        // If the value is an object, stringify it first
        if (typeof value === "object") {
          value = JSON.stringify(value);
        }

        // Escape double quotes in the value by doubling them
        if (typeof value === "string") {
          value = value.replace(/"/g, '""');
        }

        // Wrap the value in double quotes to enclose any value tha
        // might have a, or a " in it to distinguish them from a comma separated delimiter
        value = `"${value}"`;

        return value;
      })
      .join(",");
  });

  rows.unshift(jsonFields.join(","));

  return rows.join("\n");
};

export default convertToCSV;
