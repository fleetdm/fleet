import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import React from "react";
import { IMacSetting, IMacSettings } from "interfaces/mdm";
import MacSettingsIndicator from "../../MacSettingsIndicator";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface ICellProps {
  cell: {
    value: string;
  };
  row: {
    original: IMacSetting;
  };
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell: (props: ICellProps) => JSX.Element;
  id?: string;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

const SETTING_STATUS_OPTIONS = {
  "Action required (pending)": {
    iconName: "pending",
    tooltipText: "Follow Disk encryption instructions on your My device page.",
  },
  Applied: {
    iconName: "success",
    tooltipText: "Disk encryption on and disk encryption key stored in Fleet.",
  },
  "Enforcing (pending)": {
    iconName: "pending",
    tooltipText: "Setting will be enforced when the host comes online.",
  },
  "Removing enforcement (pending)": {
    iconName: "pending",
    tooltipText: "Enforcement will be removed when the host comes online.",
  },
  Failed: { iconName: "error", tooltipText: null },
} as const;

const tableHeaders: IDataColumn[] = [
  {
    title: "Name",
    Header: "Name",
    disableSortBy: true,
    accessor: "name",
    Cell: (cellProps: ICellProps): JSX.Element => (
      <TextCell value={cellProps.cell.value} />
    ),
  },
  {
    title: "Status",
    Header: "Status",
    disableSortBy: true,
    accessor: "statusText",
    Cell: (cellProps: ICellProps) => {
      // TODO: refine this logic according to API structure
      const statusData = cellProps.row.original;
      const statusText = statusData.statusText;
      // const statusText = "Applied";
      const iconName = SETTING_STATUS_OPTIONS[statusText].iconName;
      const tooltip = {
        tooltipText: SETTING_STATUS_OPTIONS[statusText].tooltipText,
        position: "bottom" as const,
      };
      return (
        <MacSettingsIndicator
          indicatorText={statusText}
          iconName={iconName}
          tooltip={tooltip}
        />
      );
    },
  },
  {
    title: "Error",
    Header: "Error",
    disableSortBy: true,
    accessor: "error",
    Cell: (cellProps: ICellProps): JSX.Element => {
      // TODO: logically generate settings error from API structure
      return <div>Error</div>;
    },
  },
];

const generateDataSet = (hostMacSettings: IMacSettings): IMacSettings => {
  // TODO - make this real
  return hostMacSettings;
};

export { tableHeaders, generateDataSet };
