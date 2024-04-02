import convertToCSV from "utilities/convert_to_csv";
import { Row, Column } from "react-table";
import { ICampaignError } from "interfaces/campaign";
import { format } from "date-fns";

const reorderCSVFields = (tableHeaders: string[]) => {
  const result = tableHeaders.filter((field) => field !== "host_display_name");
  result.unshift("host_display_name");

  return result;
};

export const generateCSVFilename = (descriptor: string) => {
  return `${descriptor} (${format(new Date(), "MM-dd-yy hh-mm-ss")}).csv`;
};

// Live query results, live query errors, and query report
export const generateCSVQueryResults = <T extends object>(
  rows: Row[],
  filename: string,
  tableHeaders: Column<T>[] | string[],
  omitHostDisplayName?: boolean
) => {
  return new global.window.File(
    [
      convertToCSV({
        objArray: rows.map((r) => r.original),
        fieldSortFunc: omitHostDisplayName ? undefined : reorderCSVFields,
        tableHeaders,
      }),
    ],
    filename,
    {
      type: "text/csv",
    }
  );
};

// Live policy results only
export const generateCSVPolicyResults = (
  rows: { host: string; status: string }[],
  filename: string
) => {
  return new global.window.File([convertToCSV({ objArray: rows })], filename, {
    type: "text/csv",
  });
};

// Live policy errors only
export const generateCSVPolicyErrors = (
  rows: ICampaignError[],
  filename: string
) => {
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
