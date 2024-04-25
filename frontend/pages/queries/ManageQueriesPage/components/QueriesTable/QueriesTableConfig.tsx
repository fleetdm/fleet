/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import ReactTooltip from "react-tooltip";
import formatDistanceToNow from "date-fns/formatDistanceToNow";
import PATHS from "router/paths";

import permissionsUtils from "utilities/permissions";
import { IUser } from "interfaces/user";
import { secondsToDhms } from "utilities/helpers";
import {
  IEnhancedQuery,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import { SupportedPlatform } from "interfaces/platform";

import Icon from "components/Icon";
import Checkbox from "components/forms/fields/Checkbox";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell/HeaderCell";
import PlatformCell from "components/TableContainer/DataTable/PlatformCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import PerformanceImpactCell from "components/TableContainer/DataTable/PerformanceImpactCell";
import TooltipWrapper from "components/TooltipWrapper";
import { COLORS } from "styles/var/colors";
import QueryAutomationsStatusIndicator from "../QueryAutomationsStatusIndicator";

interface IQueryRow {
  id: string;
  original: ISchedulableQuery;
}

interface IGetToggleAllRowsSelectedProps {
  checked: boolean;
  indeterminate: boolean;
  title: string;
  onChange: () => void;
  style: { cursor: string };
}
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
  getToggleAllRowsSelectedProps: () => IGetToggleAllRowsSelectedProps;
  toggleAllRowsSelected: () => void;
  toggleRowSelected: (id: string, value?: boolean) => void;
  rows: IQueryRow[];
  selectedFlatRows: IQueryRow[];
}
interface IRowProps {
  row: {
    original: IEnhancedQuery;
    getToggleRowSelectedProps: () => IGetToggleAllRowsSelectedProps;
    toggleRowSelected: () => void;
  };
  toggleRowSelected: (id: string, value: boolean) => void;
}

interface ICellProps extends IRowProps {
  cell: {
    value: string | number | boolean;
  };
}

interface INumberCellProps extends IRowProps {
  cell: {
    value: number;
  };
}

interface IStringCellProps extends IRowProps {
  cell: { value: string };
}

interface IBoolCellProps extends IRowProps {
  cell: { value: boolean };
}
interface IPlatformCellProps extends IRowProps {
  cell: {
    value: SupportedPlatform[];
  };
}

interface IDataColumn {
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IPlatformCellProps) => JSX.Element)
    | ((props: IStringCellProps) => JSX.Element)
    | ((props: INumberCellProps) => JSX.Element)
    | ((props: IBoolCellProps) => JSX.Element);
  id?: string;
  title?: string;
  accessor?: string;
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

interface IGenerateTableHeaders {
  currentUser: IUser;
  currentTeamId?: number;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = ({
  currentUser,
  currentTeamId,
}: IGenerateTableHeaders): IDataColumn[] => {
  const isOnlyObserver = permissionsUtils.isOnlyObserver(currentUser);

  const tableHeaders: IDataColumn[] = [
    {
      title: "Name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      Cell: (cellProps: ICellProps): JSX.Element => {
        return (
          <LinkCell
            className="w400 query-name-cell"
            value={
              <>
                <div className="query-name-text">{cellProps.cell.value}</div>
                {!isOnlyObserver && cellProps.row.original.observer_can_run && (
                  <>
                    <span
                      className="tooltip-base"
                      data-tip
                      data-for={`observer-can-run-tooltip-${cellProps.row.original.id}`}
                    >
                      <Icon className="query-icon" name="query" size="small" />
                    </span>
                    <ReactTooltip
                      className="observer-can-run-tooltip"
                      place="top"
                      type="dark"
                      effect="solid"
                      id={`observer-can-run-tooltip-${cellProps.row.original.id}`}
                      backgroundColor={COLORS["tooltip-bg"]}
                    >
                      Observers can run this query.
                    </ReactTooltip>
                  </>
                )}
              </>
            }
            path={PATHS.QUERY_DETAILS(
              cellProps.row.original.id,
              cellProps.row.original.team_id ?? currentTeamId
            )}
          />
        );
      },
      sortType: "caseInsensitive",
    },
    {
      title: "Platform",
      Header: "Compatible with",
      disableSortBy: true,
      accessor: "platforms",
      Cell: (cellProps: IPlatformCellProps): JSX.Element => {
        return <PlatformCell platforms={cellProps.row.original.platforms} />;
      },
    },
    {
      title: "Frequency",
      Header: "Frequency",
      disableSortBy: true,
      accessor: "interval",
      Cell: (cellProps: INumberCellProps): JSX.Element => {
        const val = cellProps.cell.value
          ? `Every ${secondsToDhms(cellProps.cell.value)}`
          : undefined;
        return (
          <TextCell
            value={val}
            emptyCellTooltipText={
              <>Assign a frequency to collect data at an interval.</>
            }
          />
        );
      },
    },
    {
      title: "Performance impact",
      Header: () => {
        return (
          <div>
            <TooltipWrapper tipContent="The average performance impact across all hosts.">
              Performance impact
            </TooltipWrapper>
          </div>
        );
      },
      disableSortBy: true,
      accessor: "performance",
      Cell: (cellProps: IStringCellProps) => (
        <PerformanceImpactCell
          value={{
            indicator: cellProps.cell.value,
            id: cellProps.row.original.id,
          }}
        />
      ),
    },
    {
      title: "Automations",
      Header: "Automations",
      disableSortBy: true,
      accessor: "automations_enabled",
      Cell: (cellProps: IBoolCellProps): JSX.Element => {
        return (
          <QueryAutomationsStatusIndicator
            automationsEnabled={cellProps.cell.value}
            interval={cellProps.row.original.interval}
          />
        );
      },
    },
    {
      title: "Last modified",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "updated_at",
      Cell: (cellProps: INumberCellProps): JSX.Element => (
        <TextCell
          value={formatDistanceToNow(new Date(cellProps.cell.value), {
            includeSeconds: true,
            addSuffix: true,
          })}
        />
      ),
    },
  ];
  if (!isOnlyObserver) {
    tableHeaders.splice(0, 0, {
      id: "selection",
      Header: (cellProps: IHeaderProps): JSX.Element => {
        const {
          getToggleAllRowsSelectedProps,
          toggleAllRowsSelected,
        } = cellProps;
        const { checked, indeterminate } = getToggleAllRowsSelectedProps();

        const checkboxProps = {
          value: checked,
          indeterminate,
          onChange: () => {
            toggleAllRowsSelected();
          },
        };
        return <Checkbox {...checkboxProps} />;
      },
      Cell: (cellProps: ICellProps): JSX.Element => {
        const { row } = cellProps;
        const { checked } = row.getToggleRowSelectedProps();
        const checkboxProps = {
          value: checked,
          onChange: () => row.toggleRowSelected(),
        };
        // v4.35.0 Any team admin or maintainer now can add, edit, delete their team's queries
        return <Checkbox {...checkboxProps} />;
      },
      disableHidden: true,
    });
  }
  return tableHeaders;
};

export default generateTableHeaders;
