import Kolide from 'kolide';

import config from 'redux/nodes/entities/users/config';
import { formatErrorResponse } from 'redux/nodes/entities/base/helpers';

const { extendedActions } = config;

// Actions for admin to require password reset for a user
export const REQUIRE_PASSWORD_RESET_SUCCESS = 'REQUIRE_PASSWORD_RESET_SUCCESS';
export const REQUIRE_PASSWORD_RESET_FAILURE = 'REQUIRE_PASSWORD_RESET_FAILURE';

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

export const changePassword = (user, { new_password: newPassword, old_password: oldPassword }) => {
  const { successAction, updateFailure, updateRequest, updateSuccess } = extendedActions;

  return (dispatch) => {
    dispatch(updateRequest);

    return Kolide.users.changePassword({ new_password: newPassword, old_password: oldPassword })
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

export const enableUser = (user, { enabled }) => {
  const { successAction, updateFailure, updateSuccess } = extendedActions;

  return (dispatch) => {
    return Kolide.users.enable(user, { enabled })
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
    return Kolide.requirePasswordReset(user, { require })
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
  const { successAction, updateFailure, updateSuccess } = extendedActions;

  return (dispatch) => {
    return Kolide.users.updateAdmin(user, { admin })
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

export default { ...config.actions, changePassword, enableUser, requirePasswordReset, updateAdmin };
