import { size, startsWith } from 'lodash';

const validate = (formData) => {
  const errors = {};
  const { kolide_web_address: kolideWebAddress } = formData;

  if (!kolideWebAddress) {
    errors.kolide_web_address = 'Kolide web address must be completed';
  }

  if (kolideWebAddress && !startsWith(kolideWebAddress, 'https://')) {
    errors.kolide_web_address = 'Kolide web address must start with https://';
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default { validate };
