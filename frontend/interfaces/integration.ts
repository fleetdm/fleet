export interface IJiraIntegration {
  url: string;
  username: string;
  password: string;
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
  password: string;
  projectKey: string;
  enableSoftwareVulnerabilities?: boolean;
}

export interface IJiraIntegrationFormErrors {
  url?: string | null;
  username?: string | null;
  password?: string | null;
  projectKey?: string | null;
}

export interface IIntegrations {
  jira: IJiraIntegration[];
}

export type IIntegration = IJiraIntegration;
