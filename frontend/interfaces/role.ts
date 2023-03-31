import PropTypes from "prop-types";
import { UserRole } from "./user";

export default PropTypes.shape({
  disabled: PropTypes.bool,
  label: PropTypes.string,
  value: PropTypes.any, // eslint-disable-line react/forbid-prop-types
  helpText: PropTypes.string,
});

export interface IRole {
  disabled: boolean;
  label: string;
  value: UserRole;
  helpText?: string;
}
