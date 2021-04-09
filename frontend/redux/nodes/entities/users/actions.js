import Kolide from "kolide";

import config from "redux/nodes/entities/users/config";
import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import { logoutUser, updateUserSuccess } from "redux/nodes/auth/actions";

const { actions } = config;

// Actions for admin to require password reset for a user
export const REQUIRE_PASSWORD_RESET_SUCCESS = "REQUIRE_PASSWORD_RESET_SUCCESS";
export const REQUIRE_PASSWORD_RESET_FAILURE = "REQUIRE_PASSWORD_RESET_FAILURE";

export const requirePasswordResetSuccess = (user) => {
  return {
    type: REQUIRE_PASSWORD_RESET_SUCCESS,
    payload: { user },
  };
};

export const requirePasswordResetFailure = (errors) => {
  return {
    type: REQUIRE_PASSWORD_RESET_FAILURE,
    payload: { errors },
  };
};

export const changePassword = (
  user,
  { new_password: newPassword, old_password: oldPassword }
) => {
  const {
    successAction,
    updateFailure,
    updateRequest,
    updateSuccess,
  } = actions;

  return (dispatch) => {
    dispatch(updateRequest());

    return Kolide.users
      .changePassword({ new_password: newPassword, old_password: oldPassword })
      .then(() => {
        return dispatch(successAction(user, updateSuccess));
      })
      .catch((response) => {
        const errorsObject = formatErrorResponse(response);

        dispatch(updateFailure(errorsObject));

        throw errorsObject;
      });
  };
};

export const confirmEmailChange = (user, token) => {
  const { loadRequest, successAction, updateFailure, updateSuccess } = actions;

  return (dispatch) => {
    dispatch(loadRequest());

    return Kolide.users
      .confirmEmailChange(user, token)
      .then((updatedUser) => {
        dispatch(successAction(updatedUser, updateSuccess));
        dispatch(updateUserSuccess(updatedUser));

        return updatedUser;
      })
      .catch((response) => {
        const errorsObject = formatErrorResponse(response);

        dispatch(updateFailure(errorsObject));

        return dispatch(logoutUser());
      });
  };
};

export const enableUser = (user, { enabled }) => {
  const { successAction, updateFailure, updateSuccess } = actions;

  return (dispatch) => {
    return Kolide.users
      .enable(user, { enabled })
      .then((userResponse) => {
        return dispatch(successAction(userResponse, updateSuccess));
      })
      .catch((response) => {
        const errorsObject = formatErrorResponse(response);

        dispatch(updateFailure(errorsObject));

        throw errorsObject;
      });
  };
};

export const requirePasswordReset = (user, { require }) => {
  return (dispatch) => {
    return Kolide.users
      .requirePasswordReset(user, { require })
      .then((updatedUser) => {
        dispatch(requirePasswordResetSuccess(updatedUser));

        return updatedUser;
      })
      .catch((response) => {
        const errorsObject = formatErrorResponse(response);
        dispatch(requirePasswordResetFailure(errorsObject));

        throw response;
      });
  };
};

export const updateAdmin = (user, { admin }) => {
  const { successAction, updateFailure, updateSuccess } = actions;

  return (dispatch) => {
    return Kolide.users
      .updateAdmin(user, { admin })
      .then((userResponse) => {
        return dispatch(successAction(userResponse, updateSuccess));
      })
      .catch((response) => {
        const errorsObject = formatErrorResponse(response);

        dispatch(updateFailure(errorsObject));

        throw errorsObject;
      });
  };
};

export default {
  ...actions,
  changePassword,
  confirmEmailChange,
  enableUser,
  requirePasswordReset,
  updateAdmin,
};
