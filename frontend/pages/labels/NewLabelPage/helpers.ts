import { INewLabelFormData } from "./NewLabelPage";

export interface INewLabelFormErrors {
  name?: string | null;
  labelQuery?: string | null;
  criteria?: string | null;
}

const MAX_LABEL_NAME_LENGTH = 255;

export const validateNewLabelFormData = (
  data: INewLabelFormData
): INewLabelFormErrors => {
  const errors: INewLabelFormErrors = {};
  const { name, type, labelQuery, vitalValue } = data;

  // name
  if (!name.trim()) {
    errors.name = "Label name must be present";
  } else if (name.length > MAX_LABEL_NAME_LENGTH) {
    errors.name = `Name may not exceed ${MAX_LABEL_NAME_LENGTH} characters`;
  }

  // dynamic label query
  if (type === "dynamic") {
    if (!labelQuery.trim()) {
      errors.labelQuery = "Query text must be present";
    }
  }

  // host vitals criteria
  if (type === "host_vitals") {
    if (!vitalValue.trim()) {
      errors.criteria = "Label criteria must be completed";
    }
  }

  return errors;
};
