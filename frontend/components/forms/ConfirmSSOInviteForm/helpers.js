import { size } from "lodash";

const validate = (formData) => {
  const errors = {};
  const { name } = formData;

  if (!name) {
    errors.name = "Full name must be present";
  }

  const valid = !size(errors);

  return { valid, errors };
};

export default { validate };
