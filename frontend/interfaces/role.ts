import PropTypes from "prop-types";
import { UserRole } from "./user";

export default PropTypes.shape({
  disabled: PropTypes.bool,
  label: PropTypes.string,
  value: PropTypes.any, // eslint-disable-line react/forbid-prop-types
  helpText: PropTypes.string,
});

/** roles names we use in response and requests to the API. */
export type Role =
  | "admin"
  | "maintainer"
  | "observer"
  | "observer_plus"
  | "gitops";

/** role names as they apppear displayed in the UI */
export type RoleDisplay =
  | "Admin"
  | "Maintainer"
  | "Observer"
  | "Observer+"
  | "GitOps"
  | "Unassigned"
  | "Various"
  | "";
