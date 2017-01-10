import { configSuccess } from 'redux/nodes/app/actions';
import { formatErrorResponse } from 'redux/nodes/entities/base/helpers';
import Kolide from 'kolide';

export const CLEAR_AUTH_ERRORS = 'CLEAR_AUTH_ERRORS';
export const LOGIN_REQUEST = 'LOGIN_REQUEST';
export const LOGIN_SUCCESS = 'LOGIN_SUCCESS';
export const LOGIN_FAILURE = 'LOGIN_FAILURE';
export const UPDATE_USER_REQUEST = 'UPDATE_USER_REQUEST';
export const UPDATE_USER_SUCCESS = 'UPDATE_USER_SUCCESS';
export const UPDATE_USER_FAILURE = 'UPDATE_USER_FAILURE';
export const LOGOUT_REQUEST = 'LOGOUT_REQUEST';
export const LOGOUT_SUCCESS = 'LOGOUT_SUCCESS';
export const LOGOUT_FAILURE = 'LOGOUT_FAILURE';
// Actions for a user resetting their password after a required reset
export const PERFORM_REQUIRED_PASSWORD_RESET_REQUEST = 'PERFORM_REQUIRED_PASSWORD_RESET_REQUEST';
export const PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS = 'PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS';
export const PERFORM_REQUIRED_PASSWORD_RESET_FAILURE = 'PERFORM_REQUIRED_PASSWORD_RESET_FAILURE';

export const clearAuthErrors = { type: CLEAR_AUTH_ERRORS };
export const loginRequest = { type: LOGIN_REQUEST };
export const loginSuccess = ({ user, token }) => {
  return {
    type: LOGIN_SUCCESS,
    payload: {
      user,
      token,
    },
  };
};
export const loginFailure = (errors) => {
  return {
    type: LOGIN_FAILURE,
    payload: {
      errors,
    },
  };
};

export const fetchCurrentUser = () => {
  return (dispatch) => {
    dispatch(loginRequest);
    return Kolide.me()
      .then((user) => {
        return dispatch(loginSuccess({ user }));
      })
      .catch((response) => {
        dispatch(loginFailure({ base: 'Unable to authenticate the current user' }));
        throw response;
      });
  };
};

// formData should be { username: <string>, password: <string> }
export const loginUser = (formData) => {
  return (dispatch) => {
    return new Promise((resolve, reject) => {
      dispatch(loginRequest);

      return Kolide.loginUser(formData)
        .then((response) => {
          dispatch(loginSuccess(response));

          return resolve(response.user);
        })
        .catch((response) => {
          const errorObject = formatErrorResponse(response);

          dispatch(loginFailure(errorObject));

          return reject(response);
        });
    });
  };
};

export const setup = (registrationFormData) => {
  return (dispatch) => {
    return Kolide.setup(registrationFormData)
      .then((response) => {
        return dispatch(configSuccess({
          kolide_server_url: response.kolide_server_url,
          ...response.org_info,
        }));
      });
  };
};
export const updateUserRequest = { type: UPDATE_USER_REQUEST };
export const updateUserSuccess = (user) => {
  return {
    type: UPDATE_USER_SUCCESS,
    payload: {
      user,
    },
  };
};
export const updateUserFailure = (errors) => {
  return {
    type: UPDATE_USER_FAILURE,
    payload: {
      errors,
    },
  };
};
export const updateUser = (targetUser, formData) => {
  return (dispatch) => {
    return new Promise((resolve, reject) => {
      dispatch(updateUserRequest);

      return Kolide.updateUser(targetUser, formData)
        .then((user) => {
          dispatch(updateUserSuccess(user));

          return resolve(user);
        })
        .catch((response) => {
          const errorObject = formatErrorResponse(response);

          dispatch(updateUserFailure(errorObject));

          return reject(response);
        });
    });
  };
};

export const logoutFailure = (errors) => {
  return {
    type: LOGOUT_FAILURE,
    payload: {
      errors,
    },
  };
};
export const logoutRequest = { type: LOGOUT_REQUEST };
export const logoutSuccess = { type: LOGOUT_SUCCESS };
export const logoutUser = () => {
  return (dispatch) => {
    dispatch(logoutRequest);

    return Kolide.logout()
      .then(() => dispatch(logoutSuccess))
      .catch((error) => {
        dispatch(logoutFailure({ base: 'Unable to log out of your account' }));

        throw error;
      });
  };
};

export const performRequiredPasswordResetRequest = { type: PERFORM_REQUIRED_PASSWORD_RESET_REQUEST };

export const performRequiredPasswordResetSuccess = (user) => {
  return {
    type: PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS,
    payload: { user },
  };
};

export const performRequiredPasswordResetFailure = (errors) => {
  return {
    type: PERFORM_REQUIRED_PASSWORD_RESET_FAILURE,
    payload: { errors },
  };
};

export const performRequiredPasswordReset = (resetParams) => {
  return (dispatch) => {
    dispatch(performRequiredPasswordResetRequest);

    return Kolide.performRequiredPasswordReset(resetParams)
      .then((updatedUser) => {
        dispatch(performRequiredPasswordResetSuccess(updatedUser));
      })
      .catch((response) => {
        const errorsObject = formatErrorResponse(response);
        dispatch(performRequiredPasswordResetFailure(errorsObject));

        throw response;
      });
  };
};
