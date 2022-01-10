import PropTypes from "prop-types";

export default PropTypes.shape({
  destination_url: PropTypes.string,
  policy_ids: PropTypes.arrayOf(PropTypes.number),
  enable_failing_policies_webhook: PropTypes.bool,
  host_batch_size: PropTypes.number,
});

export interface IWebhookFailingPolicies {
  destination_url?: string;
  policy_ids?: number[];
  enable_failing_policies_webhook?: boolean;
  host_batch_size?: number;
}
