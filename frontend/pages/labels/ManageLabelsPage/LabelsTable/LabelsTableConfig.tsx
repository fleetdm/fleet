import React from "react";
import { ILabel, LabelMembershipTypeToDisplayCopy } from "interfaces/label";
import { IDropdownOption } from "interfaces/dropdownOption";

import TextCell from "components/TableContainer/DataTable/TextCell";
import {
  isGlobalAdmin,
  isGlobalMaintainer,
  isAnyTeamMaintainerOrTeamAdmin,
} from "utilities/permissions/permissions";
import { IUser } from "interfaces/user";
import HeaderCell from "components/TableContainer/DataTable/HeaderCell";
import ViewAllHostsLink from "components/ViewAllHostsLink";

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
  currentUser: IUser,
  label: ILabel
): IDropdownOption[] => {
  const options: IDropdownOption[] = [
    {
      label: "View all hosts",
      disabled: false,
      value: "view_hosts",
    },
  ];

  const hasGlobalWritePermission =
    isGlobalAdmin(currentUser) || isGlobalMaintainer(currentUser);

  const hasLabelAuthorWritePermission =
    isAnyTeamMaintainerOrTeamAdmin(currentUser) &&
    label.author_id === currentUser.id;

  if (hasGlobalWritePermission || hasLabelAuthorWritePermission) {
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

const generateTableHeaders = (
  currentUser: IUser,
  onClickAction: (action: string, label: ILabel) => void
): IDataColumn[] => {
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
      Cell: (cellProps: ICellProps) => {
        const type = cellProps.row.original.label_membership_type;
        return <TextCell value={LabelMembershipTypeToDisplayCopy[type]} />;
      },
    },
    {
      title: "Actions",
      Header: "",
      disableSortBy: true,
      accessor: "actions",
      Cell: (cellProps: IDropdownCellProps) => {
        const label = cellProps.row.original;
        const dropdownOptions = generateActionDropdownOptions(
          currentUser,
          label
        );
        return (
          <ViewAllHostsLink
            rowHover
            noLink
            excludeChevron
            dropdown={{
              options: dropdownOptions,
              onChange: (value: string) => onClickAction(value, label),
            }}
          />
        );
      },
    },
  ];
};

const generateDataSet = (labels: ILabel[]) =>
  labels.filter((label) => label.label_type !== "builtin");

export { generateTableHeaders, generateDataSet };
