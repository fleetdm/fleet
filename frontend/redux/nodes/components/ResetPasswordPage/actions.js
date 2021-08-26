import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import Fleet from "fleet";

export const CLEAR_RESET_PASSWORD_ERRORS = "CLEAR_RESET_PASSWORD_ERRORS";
export const RESET_PASSWORD_ERROR = "RESET_PASSWORD_ERROR";
export const RESET_PASSWORD_REQUEST = "RESET_PASSWORD_REQUEST";
export const RESET_PASSWORD_SUCCESS = "RESET_PASSWORD_SUCCESS";

export const clearResetPasswordErrors = { type: CLEAR_RESET_PASSWORD_ERRORS };
export const resetPasswordError = (errors) => {
  return {
    type: RESET_PASSWORD_ERROR,
    payload: {
      errors,
    },
  };
};
export const resetPasswordRequest = { type: RESET_PASSWORD_REQUEST };
export const resetPasswordSuccess = { type: RESET_PASSWORD_SUCCESS };

// formData should be { new_password: <string>, password_reset_token: <string> }
export const resetPassword = (formData) => {
  return (dispatch) => {
    dispatch(resetPasswordRequest);

    return Fleet.users
      .resetPassword(formData)
      .then(() => {
        return dispatch(resetPasswordSuccess);
      })
      .catch((response) => {
        const errorsObject = formatErrorResponse(response);

        dispatch(resetPasswordError(errorsObject));

        throw response;
      });
  };
};
