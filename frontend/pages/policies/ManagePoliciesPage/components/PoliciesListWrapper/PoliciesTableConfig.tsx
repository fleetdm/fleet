/* eslint-disable react/prop-types */
// disable this rule as it was throwing an error in Header and Cell component
// definitions for the selection row for some reason when we dont really need it.
import React from "react";

// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { IPolicy } from "interfaces/policy";

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
interface IPoliciesTableData {
  name: string;
  passing: number;
  failing: number;
  id: number;
  query_id: number;
  query_name: string;
}

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
      accessor: "query_name",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Passing",
      Header: "Passing",
      disableSortBy: true,
      sortType: "caseInsensitive",
      accessor: "passing",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Failing",
      Header: "Failing",
      disableSortBy: true,
      accessor: "failing",
      Cell: (cellProps: ICellProps): JSX.Element => (
        <TextCell value={cellProps.cell.value} />
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

const generateDataSet = (all_policies: IPolicy[]): IPoliciesTableData[] => {
  //   return [...enhanceAllPoliciesData(all_policies)];
  return all_policies;
};

export { generateTableHeaders, generateDataSet };
