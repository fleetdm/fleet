// TODO: ORGANIZE INTERFACE FOR POLICY AUTOMATION 12/15

/* Config interface is a flattened version of the fleet/config API response */

import PropTypes from "prop-types";

export default PropTypes.shape({
  destination_url: PropTypes.string,
  policy_ids: PropTypes.arrayOf(PropTypes.number),
  enable_failing_policies_webhook: PropTypes.bool,
});

export interface IAutomationFormData {
  destination_url?: string;
  policy_ids?: number[];
  enable_failing_policies_webhook?: boolean;
}
