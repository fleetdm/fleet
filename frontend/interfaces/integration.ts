export interface IJiraIntegration {
  url: string;
  username: string;
  password: string;
  project_key: string;
  enable_software_vulnerabilities: boolean;
}

export interface IIntegrations {
  jira: IJiraIntegration[];
}

export type IIntegration = IJiraIntegration;
