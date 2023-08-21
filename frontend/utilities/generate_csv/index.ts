import convertToCSV from "utilities/convert_to_csv";
import { Row, Column } from "react-table";

const reorderCSVFields = (tableHeaders: string[]) => {
  const result = tableHeaders.filter((field) => field !== "host_display_name");
  result.unshift("host_display_name");

  console.log("result", result);
  return result;
};

const generateExportCSVFile = (
  rows: Row[],
  filename: string,
  tableHeaders: Column[]
) => {
  console.log("generateExportCSVFile rows", rows);
  return new global.window.File(
    [
      convertToCSV({
        objArray: rows.map((r) => r.original),
        fieldSortFunc: reorderCSVFields,
        tableHeaders,
      }),
    ],
    filename,
    {
      type: "text/csv",
    }
  );
};

export default generateExportCSVFile;
