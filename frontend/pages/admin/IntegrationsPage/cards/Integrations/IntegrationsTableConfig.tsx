import React from "react";

import TextCell from "components/TableContainer/DataTable/TextCell";
import DropdownCell from "components/TableContainer/DataTable/DropdownCell";

import {
  IJiraIntegration,
  IZendeskIntegration,
  IIntegrationTableData as IIntegrationCompleteData,
} from "interfaces/integration";
import { IDropdownOption } from "interfaces/dropdownOption";

import JiraIcon from "../../../../../../assets/images/icon-jira-24x24@2x.png";
import ZendeskIcon from "../../../../../../assets/images/icon-zendesk-32x24@2x.png";

interface IHeaderProps {
  column: {
    title: string;
    isSortedDesc: boolean;
  };
}

interface IRowProps {
  row: {
    original: IIntegrationTableData;
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

export interface IIntegrationTableData extends IIntegrationCompleteData {
  actions: IDropdownOption[];
  name: string;
}

// NOTE: cellProps come from react-table
// more info here https://react-table.tanstack.com/docs/api/useTable#cell-properties
const generateTableHeaders = (
  actionSelectHandler: (
    value: string,
    integration: IIntegrationTableData
  ) => void
): IDataColumn[] => {
  return [
    {
      title: "",
      Header: "",
      disableSortBy: true,
      sortType: "caseInsensitive",
      accessor: "type",
      Cell: (cellProps: ICellProps) => {
        return (
          <div className="logo-cell">
            <img
              src={cellProps.cell.value === "jira" ? JiraIcon : ZendeskIcon}
              alt="integration-icon"
              className={
                cellProps.cell.value === "jira" ? "jira-icon" : "zendesk-icon"
              }
            />
          </div>
        );
      },
    },
    {
      title: "Name",
      Header: "Name",
      disableSortBy: true,
      sortType: "caseInsensitive",
      accessor: "name",
      Cell: (cellProps: ICellProps) => (
        <TextCell value={cellProps.cell.value} classes="w400" />
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
          placeholder="Actions"
        />
      ),
    },
  ];
};

// NOTE: may need current user ID later for permission on actions.
const generateActionDropdownOptions = (): IDropdownOption[] => {
  return [
    {
      label: "Delete",
      disabled: false,
      value: "delete",
    },
  ];
};

const enhanceJiraData = (
  jiraIntegrations: IJiraIntegration[]
): IIntegrationTableData[] => {
  return jiraIntegrations.map((integration, index) => {
    return {
      url: integration.url,
      username: integration.username,
      apiToken: integration.api_token,
      projectKey: integration.project_key,
      enableSoftwareVulnerabilities:
        integration.enable_software_vulnerabilities,
      name: `${integration.url} - ${integration.project_key}`,
      actions: generateActionDropdownOptions(),
      originalIndex: index,
      type: "jira",
    };
  });
};

const enhanceZendeskData = (
  zendeskIntegrations: IZendeskIntegration[]
): IIntegrationTableData[] => {
  return zendeskIntegrations.map((integration, index) => {
    return {
      url: integration.url,
      email: integration.email,
      apiToken: integration.api_token,
      groupId: integration.group_id,
      enableSoftwareVulnerabilities:
        integration.enable_software_vulnerabilities,
      name: `${integration.url} - ${integration.group_id}`,
      actions: generateActionDropdownOptions(),
      originalIndex: index,
      type: "zendesk",
    };
  });
};

const combineDataSets = (
  jiraIntegrations: IJiraIntegration[],
  zendeskIntegrations: IZendeskIntegration[]
): IIntegrationTableData[] => {
  const combine = [
    ...enhanceJiraData(jiraIntegrations),
    ...enhanceZendeskData(zendeskIntegrations),
  ];
  return combine.map((integration, index) => {
    return { ...integration, tableIndex: index };
  });
};

export { generateTableHeaders, combineDataSets };
