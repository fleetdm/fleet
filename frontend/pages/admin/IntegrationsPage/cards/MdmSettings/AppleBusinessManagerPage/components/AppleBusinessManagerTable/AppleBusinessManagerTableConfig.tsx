import React from "react";
import { Column } from "react-table";

import { IMdmAbmToken } from "interfaces/mdm";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";

type IAbmTableConfig = Column<IMdmAbmToken>;
type ITableStringCellProps = IStringCellProps<IMdmAbmToken>;

type ITableHeaderProps = IHeaderProps<IMdmAbmToken>;

const DEFAULT_ACTION_OPTIONS: IDropdownOption[] = [
  { value: "editTeams", label: "Edit teams", disabled: false },
  { value: "renew", label: "Renew", disabled: false },
  { value: "delete", label: "Delete", disabled: false },
];

const generateActions = () => {
  return DEFAULT_ACTION_OPTIONS;
};

export const generateTableConfig = (
  actionSelectHandler: (value: string, team: IMdmAbmToken) => void
): IAbmTableConfig[] => {
  return [
    {
      accessor: "org_name",
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell
          value="Organization name"
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
    },
    {
      accessor: "renew_date",
      Header: "Renew date",
      disableSortBy: true,
    },
    {
      accessor: "apple_id",
      Header: "Apple ID",
      disableSortBy: true,
    },
    {
      accessor: "macos_team",
      Header: "macOS team",
      disableSortBy: true,
    },
    {
      accessor: "ios_team",
      Header: "iOS team",
      disableSortBy: true,
    },
    {
      accessor: "ipados_team",
      Header: "iPadOS team",
      disableSortBy: true,
    },
    {
      Header: "",
      disableSortBy: true,
      // the accessor here is insignificant, we just need it as its required
      // but we don't use it.
      accessor: "id",
      Cell: (cellProps) => (
        <DropdownCell
          options={generateActions()}
          onChange={(value: string) =>
            actionSelectHandler(value, cellProps.row.original)
          }
          placeholder="Actions"
        />
      ),
    },
  ];
};

export const generateTableData = (data: IMdmAbmToken[]) => {
  return data;
};
