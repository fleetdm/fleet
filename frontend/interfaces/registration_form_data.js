import { PropTypes } from 'react';

export default PropTypes.shape({
  full_name: PropTypes.string,
  username: PropTypes.string,
  password: PropTypes.string,
  password_confirmation: PropTypes.string,
  email: PropTypes.string,
  org_name: PropTypes.string,
  org_web_url: PropTypes.string,
  org_logo_url: PropTypes.string,
  kolide_web_address: PropTypes.string,
});

