import { getErrorReason, hasStatusKey } from "interfaces/errors";

const getDeleteLabelErrorMessages = (error: unknown): string => {
  // unprocessable content status. Label is used in a custom profile
  // or software target. we have to check that status exists on the error object
  // before we can access it.
  if (hasStatusKey(error) && error.status === 422) {
    return getErrorReason(error).includes("built-in")
      ? "Built-in labels can't be modified or deleted."
      : "Couldn't delete. Software uses this label as a custom target. Remove the label from the software target and try again.";
  }

  return "Could not delete label. Please try again.";
};

export default getDeleteLabelErrorMessages;
