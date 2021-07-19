import Fleet from "fleet";

import config from "redux/nodes/entities/users/config";
import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import { logoutUser, updateUserSuccess } from "redux/nodes/auth/actions";

const { actions } = config;

// Actions for admin to require password reset for a user
export const REQUIRE_PASSWORD_RESET_SUCCESS = "REQUIRE_PASSWORD_RESET_SUCCESS";
export const REQUIRE_PASSWORD_RESET_FAILURE = "REQUIRE_PASSWORD_RESET_FAILURE";
export const CREATE_USER_WITHOUT_INVITE_SUCCESS =
  "CREATE_USER_WITHOUT_INVITE_SUCCESS";
export const CREATE_USER_WITHOUT_INVITE_FAILURE =
  "CREATE_USER_WITHOUT_INVITE_FAILURE";

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

export const createUserWithoutInviteSuccess = (user) => {
  return {
    type: CREATE_USER_WITHOUT_INVITE_SUCCESS,
    payload: { user },
  };
};

export const createUserWithoutInviteFailure = (errors) => {
  return {
    type: CREATE_USER_WITHOUT_INVITE_FAILURE,
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

    return Fleet.users
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

    return Fleet.users
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

export const createUserWithoutInvitation = (formData) => {
  return (dispatch) => {
    return Fleet.users
      .createUserWithoutInvitation(formData)
      .then((response) => {
        return dispatch(createUserWithoutInviteSuccess(response));
      })
      .catch((response) => {
        const errorsObject = formatErrorResponse(response);

        dispatch(createUserWithoutInviteFailure(errorsObject));

        throw errorsObject;
      });
  };
};

export const deleteSessions = (user) => {
  const { successAction, destroyFailure, destroySuccess } = actions;

  return (dispatch) => {
    return Fleet.users
      .deleteSessions(user)
      .then((userResponse) => {
        return dispatch(successAction(userResponse, destroySuccess));
      })
      .catch((response) => {
        const errorsObject = formatErrorResponse(response);

        dispatch(destroyFailure(errorsObject));

        throw errorsObject;
      });
  };
};

export const enableUser = (user, { enabled }) => {
  const { successAction, updateFailure, updateSuccess } = actions;

  return (dispatch) => {
    return Fleet.users
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

export const requirePasswordReset = (userId, { require }) => {
  return (dispatch) => {
    return Fleet.users
      .requirePasswordReset(userId, { require })
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
    return Fleet.users
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
  createUserWithoutInvitation,
  enableUser,
  requirePasswordReset,
  deleteSessions,
  updateAdmin,
};
