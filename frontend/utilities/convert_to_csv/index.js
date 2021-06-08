import { keys } from "lodash";

const defaultFieldSortFunc = (fields) => fields;

const convertToCSV = (objArray, fieldSortFunc = defaultFieldSortFunc) => {
  const fields = fieldSortFunc(keys(objArray[0]));
  const jsonFields = fields.map((field) => JSON.stringify(field));
  const rows = objArray.map((row) => {
    return fields.map((field) => JSON.stringify(row[field])).join(",");
  });

  rows.unshift(jsonFields.join(","));

  return rows.join("\n");
};

export default convertToCSV;
