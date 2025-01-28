import React from "react";
import { CellProps, Column } from "react-table";

import { IMdmAbmToken, IMdmVppToken } from "interfaces/mdm";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ActionsDropdown from "components/ActionsDropdown";
import TextCell from "components/TableContainer/DataTable/TextCell";

import RenewDateCell from "../../../components/RenewDateCell";
import { IRenewDateCellStatusConfig } from "../../../components/RenewDateCell/RenewDateCell";
import TeamsCell from "./TeamsCell";

type IAbmTableConfig = Column<IMdmVppToken>;
type ITableStringCellProps = IStringCellProps<IMdmVppToken>;
type IRenewDateCellProps = CellProps<IMdmVppToken, IMdmVppToken["renew_date"]>;
type ITeamsCellProps = CellProps<IMdmVppToken, IMdmVppToken["teams"]>;

type ITableHeaderProps = IHeaderProps<IMdmVppToken>;

const DEFAULT_ACTION_OPTIONS: IDropdownOption[] = [
  { value: "editTeams", label: "Edit teams", disabled: false },
  { value: "renew", label: "Renew", disabled: false },
  { value: "delete", label: "Delete", disabled: false },
];

const generateActions = () => {
  return DEFAULT_ACTION_OPTIONS;
};

const RENEW_DATE_CELL_STATUS_CONFIG: IRenewDateCellStatusConfig = {
  warning: {
    tooltipText: (
      <>
        VPP content token is less than 30 days from expiration.
        <br />
        To renew, go to <b>Actions {">"} Renew</b>.
      </>
    ),
  },
  error: {
    tooltipText: (
      <>
        VPP content token is expired.
        <br />
        To renew, go to <b>Actions {">"} Renew</b>.
      </>
    ),
  },
};

export const generateTableConfig = (
  actionSelectHandler: (value: string, team: IMdmVppToken) => void
): IAbmTableConfig[] => {
  return [
    {
      accessor: "org_name",
      sortType: "caseInsensitive",
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell
          value="Organization name"
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      accessor: "location",
      Header: "Location",
      disableSortBy: true,
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      accessor: "renew_date",
      Header: "Renew date",
      disableSortBy: true,
      Cell: (cellProps: IRenewDateCellProps) => (
        <RenewDateCell
          value={cellProps.cell.value}
          statusConfig={RENEW_DATE_CELL_STATUS_CONFIG}
          className="vpp-renew-date-cell"
        />
      ),
    },

    {
      accessor: "teams",
      Header: "Teams",
      disableSortBy: true,
      Cell: (cellProps: ITeamsCellProps) => (
        <TeamsCell teams={cellProps.cell.value} className="vpp-teams-cell" />
      ),
    },
    {
      Header: "",
      disableSortBy: true,
      // the accessor here is insignificant, we just need it as its required
      // but we don't use it.
      accessor: "id",
      Cell: (cellProps) => (
        <ActionsDropdown
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
