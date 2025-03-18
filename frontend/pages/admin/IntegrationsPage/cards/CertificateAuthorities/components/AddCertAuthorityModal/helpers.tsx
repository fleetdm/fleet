const DEFAULT_ERROR_MESSAGE =
  "Couldn't add certificate authority. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const generateErrorMessage = (e: unknown) => {
  return DEFAULT_ERROR_MESSAGE;
};
