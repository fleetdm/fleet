import React from "react";
import { ILabel } from "interfaces/label";
import { IDropdownOption } from "interfaces/dropdownOption";

import TextCell from "components/TableContainer/DataTable/TextCell";
import ActionsDropdown from "components/ActionsDropdown";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import {
  isGlobalAdmin,
  isGlobalMaintainer,
} from "utilities/permissions/permissions";
import { IUser } from "interfaces/user";
import { capitalize } from "lodash";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IRowProps {
  row: {
    original: ILabel;
  };
}

interface ICellProps extends IRowProps {
  cell: {
    value: string;
  };
}

interface IDropdownCellProps extends IRowProps {
  cell: {
    value: IDropdownOption[];
  };
}

interface IDataColumn {
  title: string;
  Header: ((props: IHeaderProps) => JSX.Element) | string;
  accessor: string;
  Cell:
    | ((props: ICellProps) => JSX.Element)
    | ((props: IDropdownCellProps) => JSX.Element);
  disableHidden?: boolean;
  disableSortBy?: boolean;
  sortType?: string;
}

const generateActionDropdownOptions = (
  currentUser: IUser
): IDropdownOption[] => {
  const options: IDropdownOption[] = [
    {
      label: "View all hosts",
      disabled: false,
      value: "view_hosts",
    },
  ];

  if (isGlobalAdmin(currentUser) || isGlobalMaintainer(currentUser)) {
    options.push(
      {
        label: "Edit",
        disabled: false,
        value: "edit",
      },
      {
        label: "Delete",
        disabled: false,
        value: "delete",
      }
    );
  }

  return options;
};

// Generate table headers with action handler
const generateTableHeaders = (
  currentUser: IUser,
  onClickAction: (action: string, label: ILabel) => void
): IDataColumn[] => {
  const dropdownOptions = generateActionDropdownOptions(currentUser);

  return [
    {
      title: "Name",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "name",
      disableSortBy: false,
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Description",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "description",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value || ""} />
      ),
    },
    {
      title: "Type",
      Header: (cellProps) => (
        <HeaderCell
          value={cellProps.column.title}
          isSortedDesc={cellProps.column.isSortedDesc}
        />
      ),
      accessor: "label_membership_type",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={capitalize(cellProps.cell.value)} />
      ),
    },
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps: IDropdownCellProps) => (
        <ActionsDropdown
          options={dropdownOptions}
          onChange={(value: string) =>
            onClickAction(value, cellProps.row.original)
          }
          placeholder="Actions"
        />
      ),
    },
  ];
};

const generateDataSet = (labels: ILabel[]) =>
  labels.filter((label) => label.label_type !== "builtin");

export { generateTableHeaders, generateDataSet };
