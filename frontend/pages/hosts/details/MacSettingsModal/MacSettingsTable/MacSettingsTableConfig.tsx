import TextCell from "components/TableContainer/DataTable/TextCell";
import React from "react";
import {
  IMacMdmProfile,
  MacMdmProfileOperationType,
  MacMdmProfileStatus,
} from "interfaces/mdm";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import TruncatedTextCell from "components/TableContainer/DataTable/TruncatedTextCell";
import MacSettingsIndicator from "../../MacSettingsIndicator";
import { IMacSettingsIndicator } from "../../MacSettingsIndicator/MacSettingsIndicator";

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

const PROFILE_DISPLAY_CONFIG: Record<
  MacMdmProfileOperationType,
  Record<MacMdmProfileStatus, IMacSettingsIndicator | null>
> = {
  install: {
    pending: {
      indicatorText: "Enforcing (pending)",
      iconName: "pending",
      tooltip: {
        tooltipText: "Setting will be enforced when the host comes online.",
      },
    },
    success: {
      indicatorText: "Applied",
      iconName: "success",
      tooltip: { tooltipText: "Host applied the setting." },
    },
    failed: {
      indicatorText: "Failed",
      iconName: "error",
      tooltip: undefined,
    },
  },
  remove: {
    pending: {
      indicatorText: "Removing enforcement (pending)",
      iconName: "pending",
      tooltip: {
        tooltipText: "Enforcement will be removed when the host comes online.",
      },
    },
    success: null, // should not be reached
    failed: {
      indicatorText: "Failed",
      iconName: "error",
      tooltip: undefined,
    },
  },
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
      const { status, operation_type } = cellProps.row.original;
      const options = PROFILE_DISPLAY_CONFIG[operation_type]?.[status];
      if (options) {
        const { indicatorText, iconName } = options;
        const tooltip = {
          tooltipText: options.tooltip?.tooltipText ?? null,
          position: "top" as const,
        };
        return (
          <MacSettingsIndicator
            indicatorText={indicatorText}
            iconName={iconName}
            tooltip={tooltip}
          />
        );
      }

      // graceful error - this state should not be reached based on the API spec
      return <TextCell value="Unrecognized" />;
    },
  },
  {
    title: "Error",
    Header: "Error",
    disableSortBy: true,
    accessor: "detail",
    Cell: (cellProps: ICellProps): JSX.Element => {
      const profile = cellProps.row.original;
      return (
        <TruncatedTextCell
          value={
            (profile.status === "failed" && profile.detail) ||
            DEFAULT_EMPTY_CELL_VALUE
          }
        />
      );
    },
  },
];

export default tableHeaders;
