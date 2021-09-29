/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";
import { memoize } from "lodash";

// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import LinkCell from "components/TableContainer/DataTable/LinkCell/LinkCell";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { IPolicy } from "interfaces/policy";
import PATHS from "router/paths";
import sortUtils from "utilities/sort";
import { PolicyResponse } from "utilities/constants";

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
    original: IPolicy;
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
          title: "Query",
          Header: "Query",
          disableSortBy: true,
          accessor: "query_name",
          Cell: (cellProps: ICellProps): JSX.Element => (
            <TextCell value={cellProps.cell.value} />
          ),
        },
      ];
    default: {
      const tableHeaders: IDataColumn[] = [
        {
          title: "Query",
          Header: "Query",
          disableSortBy: true,
          accessor: "query_name",
          Cell: (cellProps: ICellProps): JSX.Element => (
            <LinkCell
              value={cellProps.cell.value}
              path={`${PATHS.URL_PREFIX}/queries/${cellProps.row.original.query_id}`}
            />
          ),
        },
        {
          title: "Passing",
          Header: "Passing",
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
          title: "Failing",
          Header: "Failing",
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

const generateDataSet = memoize((policiesList: IPolicy[] = []): IPolicy[] => {
  policiesList = policiesList.sort((a, b) =>
    sortUtils.caseInsensitiveAsc(a.query_name, b.query_name)
  );
  return policiesList;
});

export { generateTableHeaders, generateDataSet };
