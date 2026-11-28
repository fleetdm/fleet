import { getErrorReason } from "interfaces/errors";

const DEFAULT_ERROR_MESSAGE = "Could not update. Please try again.";

// The server sends no message when the Confirm email update password is wrong
// (a bare 403), so this copy is the page's own.
export const INCORRECT_PASSWORD_ERROR = "Incorrect password";

// eslint-disable-next-line import/prefer-default-export
export const getErrorMessage = (e: unknown) => {
  let errorMessage = getErrorReason(e, {
    reasonIncludes: "Cannot reuse old password",
  });

  if (!errorMessage) {
    errorMessage = DEFAULT_ERROR_MESSAGE;
  }

  return errorMessage;
};
