import { size, trim } from 'lodash';

import validJwtToken from 'components/forms/validators/valid_jwt_token';

export default ({ license }) => {
  const errors = {};

  if (!license) {
    errors.license = 'License must be present';
  }

  if (license && !validJwtToken(trim(license))) {
    errors.license = 'License syntax is not valid. Please ensure you have entered the entire license. Please contact support@kolide.co if you need assistance';
  }

  const valid = !size(errors);

  return { errors, valid };
};
