import PropTypes from "prop-types";

export default PropTypes.shape({
  idp_image_url: PropTypes.string,
  idp_name: PropTypes.string,
  sso_enabled: PropTypes.bool,
});

export interface ISSOSettings {
  idp_image_url: string;
  idp_name: string;
  sso_enabled: boolean;
}
