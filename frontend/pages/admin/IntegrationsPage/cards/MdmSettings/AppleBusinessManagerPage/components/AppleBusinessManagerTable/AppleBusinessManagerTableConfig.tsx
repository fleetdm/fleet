import React from "react";
import { CellProps, Column } from "react-table";

import { IMdmAbmToken } from "interfaces/mdm";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ActionsDropdown from "components/ActionsDropdown";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";

import RenewDateCell from "../../../components/RenewDateCell";
import OrgNameCell from "./OrgNameCell";
import { IRenewDateCellStatusConfig } from "../../../components/RenewDateCell/RenewDateCell";

type IAbmTableConfig = Column<IMdmAbmToken>;
type ITableStringCellProps = IStringCellProps<IMdmAbmToken>;
type IRenewDateCellProps = CellProps<IMdmAbmToken, IMdmAbmToken["renew_date"]>;

type ITableHeaderProps = IHeaderProps<IMdmAbmToken>;

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
        ABM server token is less than 30 days from expiration.
        <br /> To renew, go to <b>Actions {">"} Renew.</b>
      </>
    ),
  },
  error: {
    tooltipText: (
      <>
        ABM server token is expired.
        <br /> To renew, go to <b>Actions {">"} Renew</b>.
      </>
    ),
  },
};

export const generateTableConfig = (
  actionSelectHandler: (value: string, team: IMdmAbmToken) => void
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
      Cell: (cellProps: ITableStringCellProps) => {
        const { terms_expired, org_name } = cellProps.cell.row.original;
        return <OrgNameCell orgName={org_name} termsExpired={terms_expired} />;
      },
    },
    {
      accessor: "renew_date",
      Header: "Renew date",
      disableSortBy: true,
      Cell: (cellProps: IRenewDateCellProps) => (
        <RenewDateCell
          value={cellProps.cell.value}
          statusConfig={RENEW_DATE_CELL_STATUS_CONFIG}
          className="abm-renew-date-cell"
        />
      ),
    },
    {
      accessor: "apple_id",
      Header: "Apple ID",
      disableSortBy: true,
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      id: "macos_team",
      accessor: (originalRow) => originalRow.macos_team.name,
      Header: () => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={
              <>
                macOS hosts are automatically added to this team in Fleet when
                they appear in Apple Business Manager.
              </>
            }
          >
            macOS team
          </TooltipWrapper>
        );
        return <HeaderCell value={titleWithToolTip} disableSortBy />;
      },
      disableSortBy: true,
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      id: "ios_team",
      accessor: (originalRow) => originalRow.ios_team.name,
      Header: () => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={
              <>
                iOS hosts are automatically added to this team in Fleet when
                they appear in Apple Business Manager.
              </>
            }
          >
            iOS team
          </TooltipWrapper>
        );
        return <HeaderCell value={titleWithToolTip} disableSortBy />;
      },
      disableSortBy: true,
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      id: "ipados_team",
      accessor: (originalRow) => originalRow.ipados_team.name,
      Header: () => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={
              <>
                iPadOS hosts are automatically added to this team in Fleet when
                they appear in Apple Business Manager.
              </>
            }
          >
            iPadOS team
          </TooltipWrapper>
        );
        return <HeaderCell value={titleWithToolTip} disableSortBy />;
      },
      disableSortBy: true,
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.cell.value} />
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
