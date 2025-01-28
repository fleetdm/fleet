import PropTypes from "prop-types";

export default PropTypes.shape({
  destination_url: PropTypes.string,
  policy_ids: PropTypes.arrayOf(PropTypes.number),
  enable_failing_policies_webhook: PropTypes.bool,
  host_batch_size: PropTypes.number,
});

export interface IWebhookHostStatus {
  enable_host_status_webhook?: boolean;
  destination_url?: string;
  host_percentage?: number;
  days_count?: number;
}
export interface IWebhookFailingPolicies {
  destination_url?: string;
  policy_ids?: number[];
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

export type IWebhook =
  | IWebhookHostStatus
  | IWebhookFailingPolicies
  | IWebhookSoftwareVulnerabilities
  | IWebhookActivities;
