import PropTypes from "prop-types";

export default PropTypes.shape({
  action: PropTypes.string,
  pathname: PropTypes.string,
});

export interface IRedirectLocation {
  action: string;
  pathname: string;
}
