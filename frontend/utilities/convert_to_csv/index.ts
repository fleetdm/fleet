import { keys } from "lodash";

const defaultFieldSortFunc = (fields: string[]) => fields;

const convertToCSV = (
  objArray: any[],
  fieldSortFunc = defaultFieldSortFunc
) => {
  const fields = fieldSortFunc(keys(objArray[0]));
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
