// import { keys } from "lodash"; // REMOVED

const defaultFieldSortFunc = (fields: string[]) => fields;

const convertToCSV = (
  objArray: any[], // TODO: typing
  fieldSortFunc = defaultFieldSortFunc,
  tableHeaders: any // TODO: typing
) => {
  const tableHeadersStrings = tableHeaders.map((header: any) => header.title); // TODO: typing

  console.log("convertToCSV objArray wtf", objArray);
  // const fields = fieldSortFunc(keys(objArray[0])); // THIS IS WRONG, SHOULD NOT TAKE FIRST ROW ONLY
  const fields = fieldSortFunc(tableHeadersStrings);

  // TODO: 8/18 FIX HOST column on csv
  // TODO: Remove after v5 when host_hostname is removed rom API response.
  const hostNameIndex = fields.indexOf("host_hostname");
  if (hostNameIndex >= 0) {
    fields.splice(hostNameIndex, 1);
  }
  // Remove end
  const jsonFields = fields.map((field) => JSON.stringify(field));
  const rows = objArray.map((row) => {
    return fields.map((field) => JSON.stringify(row[field])).join(",");
  });

  rows.unshift(jsonFields.join(","));

  return rows.join("\n");
};

export default convertToCSV;
