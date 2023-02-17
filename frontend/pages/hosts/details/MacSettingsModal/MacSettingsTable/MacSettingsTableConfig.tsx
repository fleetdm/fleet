import TextCell from "components/TableContainer/DataTable/TextCell";
import React from "react";
import { IMacMdmProfile } from "interfaces/mdm";
import { uniqueId } from "lodash";
import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";
import TruncatedTextCell from "components/TableContainer/DataTable/TruncatedTextCell";
import MacSettingsIndicator from "../../MacSettingsIndicator";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

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
    original: IMacMdmProfile;
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

const getStatusDisplayOptions = (
  profile: IMacMdmProfile
): {
  statusText: string;
  iconName: "pending" | "success" | "error";
  tooltipText: string | null;
} => {
  const SETTING_STATUS_OPTIONS = {
    pending: {
      Enforcing: "Setting will be enforced when the host comes online.",
      "Removing enforcement":
        "Enforcement will be removed when the host comes online.",
      "": "",
    },
    applied: {
      iconName: "success",
      tooltipText: "Host applied the setting.",
    },
    failed: { iconName: "error", tooltipText: null },
  } as const;

  if (profile.status === "pending") {
    return {
      statusText: `${profile.detail} (pending)`,
      iconName: "pending",
      tooltipText: SETTING_STATUS_OPTIONS.pending[profile.detail],
    };
  }
  return {
    statusText:
      profile.status.charAt(0).toUpperCase() + profile.status.slice(1),
    iconName: SETTING_STATUS_OPTIONS[profile.status].iconName,
    tooltipText: SETTING_STATUS_OPTIONS[profile.status].tooltipText,
  };
};

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
      const { statusText, iconName, tooltipText } = getStatusDisplayOptions(
        cellProps.row.original
      );
      return (
        <MacSettingsIndicator
          indicatorText={statusText}
          iconName={iconName}
          tooltip={{ tooltipText, position: "top" }}
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
      const error = cellProps.row.original.error;
      return <TruncatedTextCell value={error || DEFAULT_EMPTY_CELL_VALUE} />;
    },
  },
];

export default tableHeaders;
