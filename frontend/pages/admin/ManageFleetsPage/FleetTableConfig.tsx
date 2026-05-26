import React from "react";

import { ITeam as IFleet } from "interfaces/team";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";

import LinkCell from "components/TableContainer/DataTable/LinkCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import ActionsDropdown from "components/ActionsDropdown";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IRowProps {
  row: {
    original: IFleet;
  };
}
interface ICellProps extends IRowProps {
  cell: {
    value: string;
  };
}

interface IDropdownCellProps extends IRowProps {
  cell: {
    value: IDropdownOption[];
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IDropdownCellProps) => JSX.Element);
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

interface IFleetTableData extends IFleet {
  actions: IDropdownOption[];
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  actionSelectHandler: (value: string, fleet: IFleet) => void
): IDataColumn[] => {
  return [
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      sortType: "caseInsensitive",
      accessor: "name",
      Cell: (cellProps: ICellProps) => (
        <LinkCell
          value={cellProps.cell.value}
          path={PATHS.FLEET_DETAILS_USERS(cellProps.row.original.id)}
        />
      ),
    },
    // TODO: need to add this info to API
    {
      title: "Hosts",
      Header: "Hosts",
      disableSortBy: true,
      accessor: "host_count",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Users",
      Header: "Users",
      disableSortBy: true,
      accessor: "user_count",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps: IDropdownCellProps) => (
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <div
              className={
                disableChildren
                  ? "disabled-by-gitops-mode fleet-actions-wrapper"
                  : "fleet-actions-wrapper"
              }
            >
              <ActionsDropdown
                options={cellProps.cell.value}
                onChange={(value: string) =>
                  actionSelectHandler(value, cellProps.row.original)
                }
                placeholder="Actions"
                disabled={disableChildren}
                variant="small-button"
              />
            </div>
          )}
        />
      ),
    },
  ];
};

// NOTE: may need current user ID later for permission on actions.
const generateActionDropdownOptions = (): IDropdownOption[] => {
  return [
    {
      label: "Rename",
      disabled: false,
      value: "rename",
    },
    {
      label: "Delete",
      disabled: false,
      value: "delete",
    },
  ];
};

const enhanceFleetData = (fleets: IFleet[]): IFleetTableData[] => {
  return Object.values(fleets).map((fleet) => {
    return {
      description: fleet.description,
      name: fleet.name,
      host_count: fleet.host_count,
      user_count: fleet.user_count,
      actions: generateActionDropdownOptions(),
      id: fleet.id,
    };
  });
};

const generateDataSet = (fleets: IFleet[]): IFleetTableData[] => {
  return [...enhanceFleetData(fleets)];
};

export { generateTableHeaders, generateDataSet };
