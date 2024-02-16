import PropTypes from "prop-types";

export default PropTypes.shape({
  name: PropTypes.string,
  password: PropTypes.string,
  password_confirmation: PropTypes.string,
  email: PropTypes.string,
  org_name: PropTypes.string,
  org_web_url: PropTypes.string,
  org_logo_url: PropTypes.string,
  fleet_web_address: PropTypes.string,
});

export interface IRegistrationFormData {
  name: string;
  password: string;
  password_confirmation: string;
  email: string;
  org_name: string;
  org_web_url: string;
  org_logo_url: string;
  fleet_web_address: string;
  server_url: string;
}

export interface IRegistrationFormErrors {
  name?: string | null;
  password?: string | null;
  password_confirmation?: string | null;
  email?: string | null;
  org_name?: string | null;
  org_web_url?: string | null;
  org_logo_url?: string | null;
  fleet_web_address?: string | null;
  server_url?: string | null;
}
