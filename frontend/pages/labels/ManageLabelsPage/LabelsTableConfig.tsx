import React from "react";
import { ILabel } from "interfaces/label";
import { IDropdownOption } from "interfaces/dropdownOption";
import PATHS from "router/paths";

import TextCell from "components/TableContainer/DataTable/TextCell";
import ActionsDropdown from "components/ActionsDropdown";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { isObserverPlus } from "utilities/permissions/permissions";
import { IUser } from "interfaces/user";

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

interface ILabelTableData extends ILabel {
  actions: IDropdownOption[];
}

// Generate table headers with action handler
const generateTableHeaders = (
  actionSelectHandler: (value: string, label: ILabel) => void,
  currentUser: IUser
): IDataColumn[] => {
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
      accessor: "label_type",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} />
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
                options={cellProps.cell.value}
                onChange={(value: string) =>
                  actionSelectHandler(value, cellProps.row.original)
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

// Generate action dropdown options based on user permissions
const generateActionDropdownOptions = (
  label: ILabel,
  currentUser: IUser
): IDropdownOption[] => {
  const isObserverPlusUser = isObserverPlus(currentUser, null);

  const options: IDropdownOption[] = [
    {
      label: "View all hosts",
      disabled: false,
      value: "view_hosts",
    },
  ];

  // Hide edit and delete options for Observer and Observer+ users
  if (!isObserverPlusUser) {
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

// Enhance label data with actions
const enhanceLabelData = (
  labels: ILabel[],
  currentUser: IUser
): ILabelTableData[] => {
  return labels.map((label) => {
    return {
      ...label,
      actions: generateActionDropdownOptions(label, currentUser),
    };
  });
};

// Generate the dataset for the table
const generateDataSet = (
  labels: ILabel[],
  currentUser: IUser
): ILabelTableData[] => {
  return [...enhanceLabelData(labels, currentUser)];
};

export { generateTableHeaders, generateDataSet };
