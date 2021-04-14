import { size } from "lodash";

import validateNumericality from "components/forms/validators/validate_numericality";

const validate = (formData) => {
  const errors = {};

  if (!formData.query_id) {
    errors.query_id = "A query must be selected";
  }

  if (!formData.interval) {
    errors.interval = "Interval must be present";
  }

  if (formData.interval && !validateNumericality(formData.interval)) {
    errors.interval = "Interval must be a number";
  }

  if (!formData.logging_type) {
    errors.logging_type = "A Logging Type must be selected";
  }

  if (formData.shard) {
    if (formData.shard < 0 || formData.shard > 100) {
      errors.shard = "Shard must be between 0 and 100";
    }
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default validate;
