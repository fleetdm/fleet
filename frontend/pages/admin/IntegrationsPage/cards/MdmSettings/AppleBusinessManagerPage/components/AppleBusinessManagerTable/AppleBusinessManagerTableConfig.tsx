import React from "react";
import { CellProps, Column } from "react-table";

import ActionsDropdown from "components/ActionsDropdown";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import TooltipWrapper from "components/TooltipWrapper";
import { IHeaderProps, IStringCellProps } from "interfaces/datatable_config";
import { IDropdownOption } from "interfaces/dropdownOption";
import { IMdmAbToken } from "interfaces/mdm";
import { getFleetDisplayName } from "interfaces/team";
import { getGitOpsModeTipContent } from "utilities/helpers";

import RenewDateCell from "../../../components/RenewDateCell";
import { IRenewDateCellStatusConfig } from "../../../components/RenewDateCell/RenewDateCell";

import OrgNameCell from "./OrgNameCell";

type IAbmTableConfig = Column<IMdmAbToken>;
type ITableStringCellProps = IStringCellProps<IMdmAbToken>;
type IRenewDateCellProps = CellProps<IMdmAbToken, IMdmAbToken["renew_date"]>;

type ITableHeaderProps = IHeaderProps<IMdmAbToken>;

export const generateActions = (
  token: IMdmAbToken,
  tokensCount: number,
  gitopsModeEnabled: boolean,
  repoURL?: string
): IDropdownOption[] => {
  const gitOpsDisabledProps = {
    disabled: true,
    ...(repoURL ? { tooltipContent: getGitOpsModeTipContent(repoURL) } : {}),
  };

  let toggleDefaultOption: IDropdownOption = {
    value: "toggleDefault",
    label: token.default ? "Unset default token" : "Set as default token",
    disabled: false,
  };
  if (gitopsModeEnabled) {
    toggleDefaultOption = { ...toggleDefaultOption, ...gitOpsDisabledProps };
  } else if (token.default && tokensCount === 1) {
    // the backend rejects unsetting a lone token's default
    toggleDefaultOption = {
      ...toggleDefaultOption,
      disabled: true,
      tooltipContent: "The only AB token is always the default.",
    };
  }

  return [
    {
      value: "editTeams",
      label: "Edit fleets",
      disabled: false,
      ...(gitopsModeEnabled ? gitOpsDisabledProps : {}),
    },
    toggleDefaultOption,
    { value: "renew", label: "Renew", disabled: false },
    { value: "delete", label: "Delete", disabled: false },
  ];
};

const RENEW_DATE_CELL_STATUS_CONFIG: IRenewDateCellStatusConfig = {
  warning: {
    tooltipText: (
      <>
        AB server token is less than 30 days from expiration.
        <br /> To renew, go to <b>Actions {">"} Renew.</b>
      </>
    ),
  },
  error: {
    tooltipText: (
      <>
        AB server token is expired.
        <br /> To renew, go to <b>Actions {">"} Renew</b>.
      </>
    ),
  },
};

export const generateTableConfig = (
  actionSelectHandler: (value: string, team: IMdmAbToken) => void,
  tokensCount: number,
  gitopsModeEnabled: boolean,
  repoURL?: string
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
        const {
          terms_expired,
          org_name,
          default: isDefault,
        } = cellProps.cell.row.original;
        return (
          <OrgNameCell
            orgName={org_name}
            termsExpired={terms_expired}
            isDefault={isDefault}
          />
        );
      },
    },
    {
      accessor: "renew_date",
      sortType: "dateStrings",
      Header: (cellProps: ITableHeaderProps) => (
        <HeaderCell
          value="Renew date"
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      Cell: (cellProps: IRenewDateCellProps) => (
        <RenewDateCell
          value={cellProps.cell.value}
          statusConfig={RENEW_DATE_CELL_STATUS_CONFIG}
          className="abm-renew-date-cell"
        />
      ),
    },
    {
      id: "macos_team",
      sortType: "caseInsensitive",
      accessor: (originalRow) => getFleetDisplayName(originalRow.macos_fleet),
      Header: (cellProps: ITableHeaderProps) => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={
              <>
                macOS hosts are automatically added to this fleet on initial
                sync from AB. If a host is manually assigned to a different
                fleet before enrollment, it will enroll to the newly assigned
                fleet and not the default.
              </>
            }
          >
            macOS fleet
          </TooltipWrapper>
        );
        return (
          <HeaderCell
            value={titleWithToolTip}
            isSortedDesc={cellProps.column.isSortedDesc}
          />
        );
      },
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      id: "ios_team",
      sortType: "caseInsensitive",
      accessor: (originalRow) => getFleetDisplayName(originalRow.ios_fleet),
      Header: (cellProps: ITableHeaderProps) => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={
              <>
                iOS hosts are automatically added to this fleet on initial sync
                from AB. If a host is manually assigned to a different fleet
                before enrollment, it will enroll to the newly assigned fleet
                and not the default.
              </>
            }
          >
            iOS fleet
          </TooltipWrapper>
        );
        return (
          <HeaderCell
            value={titleWithToolTip}
            isSortedDesc={cellProps.column.isSortedDesc}
          />
        );
      },
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      id: "ipados_team",
      sortType: "caseInsensitive",
      accessor: (originalRow) => getFleetDisplayName(originalRow.ipados_fleet),
      Header: (cellProps: ITableHeaderProps) => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={
              <>
                iPadOS hosts are automatically added to this fleet on initial
                sync from AB. If a host is manually assigned to a different
                fleet before enrollment, it will enroll to the newly assigned
                fleet and not the default.
              </>
            }
          >
            iPadOS fleet
          </TooltipWrapper>
        );
        return (
          <HeaderCell
            value={titleWithToolTip}
            isSortedDesc={cellProps.column.isSortedDesc}
          />
        );
      },
      Cell: (cellProps: ITableStringCellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      id: "byod_team",
      sortType: "caseInsensitive",
      accessor: (originalRow) => getFleetDisplayName(originalRow.byod_fleet),
      Header: (cellProps: ITableHeaderProps) => {
        const titleWithToolTip = (
          <TooltipWrapper
            tipContent={
              <>
                iOS/iPadOS hosts that enroll via Managed Apple Account are
                automatically added to this fleet.
              </>
            }
          >
            BYOD fleet
          </TooltipWrapper>
        );
        return (
          <HeaderCell
            value={titleWithToolTip}
            isSortedDesc={cellProps.column.isSortedDesc}
          />
        );
      },
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
        <div className="abm-actions-wrapper">
          <ActionsDropdown
            options={generateActions(
              cellProps.row.original,
              tokensCount,
              gitopsModeEnabled,
              repoURL
            )}
            onChange={(value: string) =>
              actionSelectHandler(value, cellProps.row.original)
            }
            placeholder="Actions"
            disabled={false}
            variant="secondary"
          />
        </div>
      ),
    },
  ];
};

export const generateTableData = (data: IMdmAbToken[]) => {
  return data;
};
