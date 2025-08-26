import React from "react";
import { format } from "date-fns";

import { ScriptBatchHostStatus } from "interfaces/script";
import { IScriptBatchHostResult } from "services/entities/scripts";

import TextCell from "components/TableContainer/DataTable/TextCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import StatusIndicatorWithIcon from "components/StatusIndicatorWithIcon";
import { IndicatorStatus } from "components/StatusIndicatorWithIcon/StatusIndicatorWithIcon";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface ICellProps {
  cell: {
    value: any;
  };
}

type IDataColumn = {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  Cell: (props: ICellProps) => JSX.Element;
};

// Define the columns for different statuses
const getColumnsForStatus = (status: ScriptBatchHostStatus): IDataColumn[] => {
  // Base column that's always present
  const baseColumns: IDataColumn[] = [
    {
      title: "Host name",
      Header: (cellProps: IHeaderProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "hostName",
      Cell: ({ cell: { value } }: ICellProps) => <TextCell value={value} />,
    },
  ];

  // Additional columns for specific statuses
  if (status === "ran" || status === "errored" || status === "pending") {
    baseColumns.push(
      {
        title: "Time",
        Header: (cellProps: IHeaderProps) => (
          <HeaderCell
            value={cellProps.column.title}
            isSortedDesc={cellProps.column.isSortedDesc}
          />
        ),
        accessor: "time",
        Cell: ({ cell: { value } }: ICellProps) => <TextCell value={value} />,
      },
      {
        title: "Script output",
        Header: (cellProps: IHeaderProps) => (
          <HeaderCell
            value={cellProps.column.title}
            isSortedDesc={cellProps.column.isSortedDesc}
          />
        ),
        accessor: "scriptOutput",
        Cell: ({ cell: { value } }: ICellProps) => <TextCell value={value} />,
      }
    );
  }

  return baseColumns;
};

export const generateTableHeaders = (
  status: ScriptBatchHostStatus
): IDataColumn[] => {
  return getColumnsForStatus(status);
};

export const generateTableData = (
  data: IScriptBatchHostResult[],
  status: ScriptBatchHostStatus
) => {
  if (!data || !data.length) return [];

  return data.map((host) => {
    const baseData = {
      id: host.id,
      hostName: host.display_name,
    };

    // Add additional data for specific statuses
    if (status === "ran" || status === "errored" || status === "pending") {
      return {
        ...baseData,
        time: host.script_executed_at
          ? format(new Date(host.script_executed_at), "MMM d, yyyy h:mm a")
          : "—",
        scriptOutput: host.script_output_preview || "—",
      };
    }

    return baseData;
  });
};
