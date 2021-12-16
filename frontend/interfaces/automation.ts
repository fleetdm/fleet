// TODO: ORGANIZE INTERFACE FOR POLICY AUTOMATION 12/15

/* Config interface is a flattened version of the fleet/config API response */

import PropTypes from "prop-types";

export default PropTypes.shape({
  url: PropTypes.string,
});

export interface IAutomationFormData {
  url: string;
}
