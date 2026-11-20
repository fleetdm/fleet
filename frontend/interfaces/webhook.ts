export interface IWebhookHostStatus {
  enable_host_status_webhook?: boolean;
  destination_url?: string;
  host_percentage?: number;
  days_count?: number;
}
export interface IWebhookFailingPolicies {
  destination_url?: string;
  enable_failing_policies_webhook?: boolean;
  host_batch_size?: number;
}

export interface IWebhookSoftwareVulnerabilities {
  destination_url?: string;
  enable_vulnerabilities_webhook?: boolean;
  host_batch_size?: number;
}

export interface IWebhookActivities {
  enable_activities_webhook: boolean;
  destination_url: string;
}

export interface IWebhookHostActivities {
  enable_host_activities_webhook: boolean;
  destination_url: string;
}
