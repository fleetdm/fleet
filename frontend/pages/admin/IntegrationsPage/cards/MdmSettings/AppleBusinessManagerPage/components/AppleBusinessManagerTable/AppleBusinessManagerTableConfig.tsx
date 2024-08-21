import React from "react";
import { CellProps, Column } from "react-table";

import { IMdmAbmToken } from "interfaces/mdm";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";

import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

import RenewDateCell from "../../../components/RenewDateCell";
import OrgNameCell from "./OrgNameCell";

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
      accessor: "macos_team",
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
      accessor: "ios_team",
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
      accessor: "ipados_team",
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
