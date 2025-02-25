export type IIntegrationType = "jira" | "zendesk";
export interface IJiraIntegration {
  url: string;
  username: string;
  api_token: string;
  project_key: string;
  enable_failing_policies?: boolean;
  enable_software_vulnerabilities?: boolean;
}

export interface IZendeskIntegration {
  url: string;
  email: string;
  api_token: string;
  group_id: number;
  enable_failing_policies?: boolean;
  enable_software_vulnerabilities?: boolean;
}

export interface IScepIntegration {
  url: string;
  admin_url: string;
  username: string;
  password: string;
}

export interface IIntegration {
  url: string;
  username?: string;
  email?: string;
  api_token: string;
  project_key?: string;
  group_id?: number;
  enable_failing_policies?: boolean;
  enable_software_vulnerabilities?: boolean;
  originalIndex?: number;
  type?: IIntegrationType;
  tableIndex?: number;
  dropdownIndex?: number;
  name?: string;
}

export interface IIntegrationFormData {
  url: string;
  username?: string;
  email?: string;
  apiToken: string;
  projectKey?: string;
  groupId?: number;
  enableSoftwareVulnerabilities?: boolean;
}

export interface IIntegrationTableData extends IIntegrationFormData {
  originalIndex: number;
  type: IIntegrationType;
  tableIndex?: number;
  name: string;
}

export interface IIntegrationFormErrors {
  url?: string | null;
  email?: string | null;
  username?: string | null;
  apiToken?: string | null;
  groupId?: number | null;
  projectKey?: string | null;
  enableSoftwareVulnerabilities?: boolean;
}

export interface IGlobalCalendarIntegration {
  domain: string;
  api_key_json: string;
}

interface ITeamCalendarSettings {
  enable_calendar_events: boolean;
  webhook_url: string;
}

// zendesk and jira fields are coupled – if one is present, the other needs to be present. If
// one is present and the other is null/missing, the other will be nullified. google_calendar is
// separated – it can be present without the other 2 without nullifying them.
// TODO:  Update these types to reflect this.

export interface IZendeskJiraIntegrations {
  zendesk: IZendeskIntegration[];
  jira: IJiraIntegration[];
}

// reality is that IZendeskJiraIntegrations are optional – should be something like `extends
// Partial<IZendeskJiraIntegrations>`, but that leads to a mess of types to resolve.
export interface IGlobalIntegrations extends IZendeskJiraIntegrations {
  google_calendar?: IGlobalCalendarIntegration[] | null;
  ndes_scep_proxy?: IScepIntegration | null;
}

export interface ITeamIntegrations extends IZendeskJiraIntegrations {
  google_calendar?: ITeamCalendarSettings | null;
}
