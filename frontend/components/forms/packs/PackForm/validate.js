import { size } from "lodash";

const validate = (formData) => {
  const errors = {};

  if (!formData.name) {
    errors.name = "Title field must be completed";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default validate;
