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
  hostsByStatusRoute: (id: number, status: PolicyResponse) => {
    return `?policy_id=${id}&policy_response=${status}`;
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
// interface IPoliciesTableData {
//   name: string;
//   passing: number;
//   failing: number;
//   id: number;
//   query_id: number;
//   query_name: string;
// }

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (): IDataColumn[] => {
  return [
    {
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
    },
    {
      title: "Query",
      Header: "Query",
      disableSortBy: true,
      // sortType: "caseInsensitive",
      accessor: "query_name",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
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
            TAGGED_TEMPLATES.hostsByStatusRoute(
              cellProps.row.original.id,
              PolicyResponse.PASSING
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
            TAGGED_TEMPLATES.hostsByStatusRoute(
              cellProps.row.original.id,
              PolicyResponse.FAILING
            )
          }
        />
      ),
    },
  ];
};

// const enhanceAllPoliciesData = (
//   all_policies: IPolicy[],
//   teamId: number
// ): IAllPoliciesTableData[] => {
//   return all_policies.map((policy: IPolicy) => {
//     return {
//       name: policy.name,
//       passing: policy.passing,
//       failing: policy.failing,
//       id: policy.id,
//       query_id: policy.query_id,
//     };
//   });
// };

// const generateDataSet = memoize((all_policies: IPolicy[]): IPolicy[] => {
//   all_policies = all_policies.sort((a, b) =>
//     sortUtils.caseInsensitiveAsc(b.query_name, a.query_name)
//   );
//   //   return [...enhanceAllPoliciesData(all_policies)];
//   return all_policies;
// });

const generateDataSet = memoize((all_policies: IPolicy[] = []): IPolicy[] => {
  all_policies = all_policies.sort((a, b) =>
    sortUtils.caseInsensitiveAsc(b.query_name, a.query_name)
  );
  //   return [...enhanceAllPoliciesData(all_policies)];
  return all_policies;
});

export { generateTableHeaders, generateDataSet };
