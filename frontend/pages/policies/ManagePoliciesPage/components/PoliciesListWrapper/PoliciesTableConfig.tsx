/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { memoize } from "lodash";

// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import { IPolicyStats } from "interfaces/policy";
import PATHS from "router/paths";
import sortUtils from "utilities/sort";
import { PolicyResponse } from "utilities/constants";
import PassIcon from "../../../../../../assets/images/icon-check-circle-green-16x16@2x.png";
import FailIcon from "../../../../../../assets/images/icon-exclamation-circle-red-16x16@2x.png";

// TODO functions for paths math e.g., path={PATHS.MANAGE_HOSTS + getParams(cellProps.row.original)}

const TAGGED_TEMPLATES = {
  hostsByPolicyRoute: (
    policyId: number,
    policyResponse: PolicyResponse,
    teamId: number | undefined | null
  ) => {
    return `?policy_id=${policyId}&policy_response=${policyResponse}${
      teamId ? `&team_id=${teamId}` : ""
    }`;
  },
};
interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
  getToggleAllRowsSelectedProps: () => any; // TODO: do better with types
  toggleAllRowsSelected: () => void;
}

interface ICellProps {
  cell: {
    value: any;
  };
  row: {
    original: IPolicyStats;
    getToggleRowSelectedProps: () => any; // TODO: do better with types
    toggleRowSelected: () => void;
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

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (options: {
  selectedTeamId: number | undefined | null;
  showSelectionColumn: boolean | undefined;
  tableType: string | undefined;
}): IDataColumn[] => {
  const { selectedTeamId, tableType, showSelectionColumn } = options;

  switch (tableType) {
    case "inheritedPolicies":
      return [
        {
          title: "Name",
          Header: "Name",
          disableSortBy: true,
          accessor: "name",
          Cell: (cellProps: ICellProps): JSX.Element => (
            <LinkCell
              value={cellProps.cell.value}
              path={PATHS.EDIT_POLICY(cellProps.row.original)}
            />
          ),
        },
      ];
    default: {
      const tableHeaders: IDataColumn[] = [
        {
          title: "Name",
          Header: "Name",
          disableSortBy: true,
          accessor: "name",
          Cell: (cellProps: ICellProps): JSX.Element => (
            <LinkCell
              value={cellProps.cell.value}
              path={PATHS.EDIT_POLICY(cellProps.row.original)}
            />
          ),
        },
        {
          title: "Yes",
          Header: () => (
            <>
              <img alt="host passing" src={PassIcon} />
              <span className="header-icon-text">Yes</span>
            </>
          ),
          disableSortBy: true,
          accessor: "passing_host_count",
          Cell: (cellProps: ICellProps): JSX.Element => (
            <LinkCell
              value={`${cellProps.cell.value} hosts`}
              path={
                PATHS.MANAGE_HOSTS +
                TAGGED_TEMPLATES.hostsByPolicyRoute(
                  cellProps.row.original.id,
                  PolicyResponse.PASSING,
                  selectedTeamId
                )
              }
            />
          ),
        },
        {
          title: "No",
          Header: () => (
            <>
              <img alt="host passing" src={FailIcon} />
              <span className="header-icon-text">No</span>
            </>
          ),
          disableSortBy: true,
          accessor: "failing_host_count",
          Cell: (cellProps: ICellProps): JSX.Element => (
            <LinkCell
              value={`${cellProps.cell.value} hosts`}
              path={
                PATHS.MANAGE_HOSTS +
                TAGGED_TEMPLATES.hostsByPolicyRoute(
                  cellProps.row.original.id,
                  PolicyResponse.FAILING,
                  selectedTeamId
                )
              }
            />
          ),
        },
      ];
      if (showSelectionColumn) {
        tableHeaders.splice(0, 0, {
          id: "selection",
          Header: (cellProps: IHeaderProps): JSX.Element => {
            const props = cellProps.getToggleAllRowsSelectedProps();
            const checkboxProps = {
              value: props.checked,
              indeterminate: props.indeterminate,
              onChange: () => cellProps.toggleAllRowsSelected(),
            };
            return <Checkbox {...checkboxProps} />;
          },
          Cell: (cellProps: ICellProps): JSX.Element => {
            const props = cellProps.row.getToggleRowSelectedProps();
            const checkboxProps = {
              value: props.checked,
              onChange: () => cellProps.row.toggleRowSelected(),
            };
            return <Checkbox {...checkboxProps} />;
          },
          disableHidden: true,
        });
      }
      return tableHeaders;
    }
  }
};

const generateDataSet = memoize(
  (policiesList: IPolicyStats[] = []): IPolicyStats[] => {
    policiesList = policiesList.sort((a, b) =>
      sortUtils.caseInsensitiveAsc(a.name, b.name)
    );
    return policiesList;
  }
);

export { generateTableHeaders, generateDataSet };
