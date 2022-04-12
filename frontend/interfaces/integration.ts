export interface IJiraIntegration {
  url: string;
  username: string;
  api_token: string;
  project_key: string;
  enable_software_vulnerabilities?: boolean;
  index?: number;
}

export interface IJiraIntegrationIndexed extends IJiraIntegration {
  index: number;
}

export interface IJiraIntegrationFormData {
  url: string;
  username: string;
  apiToken: string;
  projectKey: string;
  enableSoftwareVulnerabilities?: boolean;
}

export interface IJiraIntegrationFormErrors {
  url?: string | null;
  username?: string | null;
  apiToken?: string | null;
  projectKey?: string | null;
}

export interface IIntegrations {
  jira: IJiraIntegration[];
}

export type IIntegration = IJiraIntegration;
