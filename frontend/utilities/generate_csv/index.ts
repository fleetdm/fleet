import convertToCSV from "utilities/convert_to_csv";
import { Row, Column } from "react-table";
import { ICampaignError } from "interfaces/campaign";
import { format } from "date-fns";

const reorderCSVFields = (tableHeaders: string[]) => {
  const result = tableHeaders.filter((field) => field !== "host_display_name");
  result.unshift("host_display_name");

  console.log("result", result);
  return result;
};

export const generateCSVFilename = (descriptor: string) => {
  return `${descriptor} (${format(new Date(), "MM-dd-yy hh-mm-ss")}).csv`;
};

// Query results and query errors
export const generateCSVQueryResults = (
  rows: Row[],
  // | ICampaignError[],
  filename: string,
  tableHeaders: Column[] | string[]
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

// Policy results only
export const generateCSVPolicyResults = (
  rows: { host: string; status: string }[],
  filename: string
) => {
  console.log("generateExportCSVFile rows", rows);
  return new global.window.File(
    [
      convertToCSV({
        objArray: rows,
        fieldSortFunc: reorderCSVFields,
      }),
    ],
    filename,
    {
      type: "text/csv",
    }
  );
};

// Policy errors only
export const generateCSVPolicyErrors = (
  rows: ICampaignError[],
  filename: string
) => {
  console.log("generateExportCSVFile rows", rows);
  return new global.window.File([convertToCSV({ objArray: rows })], filename, {
    type: "text/csv",
  });
};

export default {
  generateCSVFilename,
  generateCSVQueryResults,
  generateCSVPolicyResults,
  generateCSVPolicyErrors,
};
