import { size } from "lodash";

import validateNumericality from "components/forms/validators/validate_numericality";

const validate = (formData) => {
  const errors = {};

  if (!formData.query_id) {
    errors.query_id = "A query must be selected";
  }

  if (!formData.interval) {
    errors.interval = "Frequency (seconds) must be present";
  }

  if (formData.interval && !validateNumericality(formData.interval)) {
    errors.interval = "Frequency must be a number";
  }

  // logging_type does not need to be validated because it is defaulted "snapshot" if unspecified.

  if (formData.shard) {
    if (formData.shard < 0 || formData.shard > 100) {
      errors.shard = "Shard must be between 0 and 100";
    }
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default validate;
