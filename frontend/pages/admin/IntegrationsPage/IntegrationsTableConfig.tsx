import React from "react";

import TextCell from "components/TableContainer/DataTable/TextCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";

import { IIntegrations, IJiraIntegration } from "interfaces/integration";
import { IDropdownOption } from "interfaces/dropdownOption";

import JiraIcon from "../../../../assets/images/icon-jira-24x24@2x.png";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IRowProps {
  row: {
    original: IJiraIntegration;
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

export interface IIntegrationTableData extends IJiraIntegration {
  actions: IDropdownOption[];
  name: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  actionSelectHandler: (value: string, integration: IJiraIntegration) => void
): IDataColumn[] => {
  return [
    {
      title: "",
      Header: "",
      disableSortBy: true,
      sortType: "caseInsensitive",
      accessor: "hosts",
      Cell: () => <img src={JiraIcon} alt="jira-icon" />,
    },
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      sortType: "caseInsensitive",
      accessor: "name",
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
        <DropdownCell
          options={cellProps.cell.value}
          onChange={(value: string) =>
            actionSelectHandler(value, cellProps.row.original)
          }
          placeholder={"Actions"}
        />
      ),
    },
  ];
};

// NOTE: may need current user ID later for permission on actions.
const generateActionDropdownOptions = (): IDropdownOption[] => {
  return [
    {
      label: "Edit",
      disabled: false,
      value: "edit",
    },
    {
      label: "Delete",
      disabled: false,
      value: "delete",
    },
  ];
};

const enhanceIntegrationData = (
  integrations: IJiraIntegration[]
): IIntegrationTableData[] => {
  return Object.values(integrations).map((integration) => {
    return {
      url: integration.url,
      username: integration.username,
      password: integration.password,
      project_key: integration.project_key,
      actions: generateActionDropdownOptions(),
      enable_software_vulnerabilities:
        integration.enable_software_vulnerabilities,
      name: `${integration.url} - ${integration.project_key}`,
      integrationIndex: integration.integrationIndex,
    };
  });
};

const generateDataSet = (
  integrations: IJiraIntegration[]
): IIntegrationTableData[] => {
  return [...enhanceIntegrationData(integrations)];
};

export { generateTableHeaders, generateDataSet };
