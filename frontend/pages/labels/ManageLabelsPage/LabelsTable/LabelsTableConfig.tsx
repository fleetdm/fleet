import React from "react";
import { ILabel } from "interfaces/label";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";

import TextCell from "components/TableContainer/DataTable/TextCell";
import ActionsDropdown from "components/ActionsDropdown";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import {
  isGlobalAdmin,
  isGlobalMaintainer,
  isObserverPlus,
} from "utilities/permissions/permissions";
import { IUser } from "interfaces/user";
import { InjectedRouter } from "react-router";
import { capitalize } from "lodash";

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
      Header: "Name",
      accessor: "name",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
      ),
    },
    {
      title: "Description",
      Header: "Description",
      accessor: "description",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value || ""} />
      ),
    },
    {
      title: "Type",
      Header: "Type",
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
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <div className={disableChildren ? "disabled-by-gitops-mode" : ""}>
              <ActionsDropdown
                options={dropdownOptions}
                onChange={(value: string) =>
                  onClickAction(value, cellProps.row.original)
                }
                placeholder="Actions"
                disabled={disableChildren}
              />
            </div>
          )}
        />
      ),
    },
  ];
};

const generateDataSet = (labels: ILabel[]) =>
  labels.filter((label) => label.label_type !== "builtin");

export { generateTableHeaders, generateDataSet };
