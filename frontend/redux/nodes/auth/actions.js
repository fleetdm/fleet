import { configSuccess } from "redux/nodes/app/actions";
import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import Fleet from "fleet";
import local from "utilities/local";
import userActions from "redux/nodes/entities/users/actions";

export const CLEAR_AUTH_ERRORS = "CLEAR_AUTH_ERRORS";
export const LOGIN_REQUEST = "LOGIN_REQUEST";
export const LOGIN_SUCCESS = "LOGIN_SUCCESS";
export const LOGIN_FAILURE = "LOGIN_FAILURE";
export const UPDATE_USER_SUCCESS = "UPDATE_USER_SUCCESS";
export const UPDATE_USER_FAILURE = "UPDATE_USER_FAILURE";
export const LOGOUT_REQUEST = "LOGOUT_REQUEST";
export const LOGOUT_SUCCESS = "LOGOUT_SUCCESS";
export const LOGOUT_FAILURE = "LOGOUT_FAILURE";
// Actions for a user resetting their password after a required reset
export const PERFORM_REQUIRED_PASSWORD_RESET_REQUEST =
  "PERFORM_REQUIRED_PASSWORD_RESET_REQUEST";
export const PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS =
  "PERFORM_REQUIRED_PASSWORD_RESET_SUCCESS";
export const PERFORM_REQUIRED_PASSWORD_RESET_FAILURE =
  "PERFORM_REQUIRED_PASSWORD_RESET_FAILURE";

export const SSO_REDIRECT_REQUEST = "SSO_REDIRECT_REQUEST";
export const SSO_REDIRECT_SUCCESS = "SSO_REDIRECT_SUCCESS";
export const SSO_REDIRECT_FAILURE = "SSO_REDIRECT_FAILURE";
export const SSO_SETTINGS_REQUEST = "SSO_SETTINGS_REQUEST";
export const SSO_SETTINGS_SUCCESS = "SSO_SETTINGS_SUCCESS";
export const SSO_SETTINGS_FAILURE = "SSO_SETTINGS_FAILURE";

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
    return Fleet.users
      .me()
      .then((user) => {
        return dispatch(loginSuccess({ user }));
      })
      .catch((response) => {
        dispatch(
          loginFailure({ base: "Unable to authenticate the current user" })
        );
        throw response;
      });
  };
};

export const ssoRedirectRequest = { type: SSO_REDIRECT_REQUEST };
export const ssoRedirectSuccess = (redirectURL) => {
  return {
    type: SSO_REDIRECT_SUCCESS,
    payload: {
      ssoRedirectURL: redirectURL,
    },
  };
};
export const ssoRedirectFailure = ({ errors }) => {
  return {
    type: SSO_REDIRECT_FAILURE,
    payload: {
      errors,
    },
  };
};
// formData { relay_url: 'some/url'}
export const ssoRedirect = (formData) => {
  return (dispatch) => {
    dispatch(ssoRedirectRequest);
    return Fleet.sessions
      .initializeSSO(formData)
      .then((response) => {
        return dispatch(ssoRedirectSuccess(response.url));
      })
      .catch((response) => {
        dispatch(
          ssoRedirectFailure({
            base: "Unable to authenticate the current user",
          })
        );
        throw response;
      });
  };
};

export const ssoSettingsRequest = { type: SSO_SETTINGS_REQUEST };
export const ssoSettingsSuccess = (settings) => {
  return {
    type: SSO_SETTINGS_SUCCESS,
    payload: {
      ssoSettings: settings,
    },
  };
};
export const ssoSettingsFailure = ({ errors }) => {
  return {
    type: SSO_SETTINGS_FAILURE,
    payload: {
      errors,
    },
  };
};

export const ssoSettings = () => {
  return (dispatch) => {
    dispatch(ssoSettingsRequest);
    return Fleet.sessions
      .ssoSettings()
      .then((response) => {
        return dispatch(ssoSettingsSuccess(response.settings));
      })
      .catch((response) => {
        dispatch(
          ssoSettingsFailure({
            base: "Unable to fetch single sign on settings",
          })
        );
        throw response;
      });
  };
};

// formData should be { email: <string>, password: <string> }
export const loginUser = (formData) => {
  return (dispatch) => {
    return new Promise((resolve, reject) => {
      dispatch(loginRequest);

      return Fleet.sessions
        .create(formData)
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
    return Fleet.account.create(registrationFormData).then((response) => {
      const { token } = response;

      dispatch(
        configSuccess({
          server_url: response.server_url,
          ...response.org_info,
        })
      );

      local.setItem("auth_token", token);
      Fleet.setBearerToken(token);

      return dispatch(fetchCurrentUser());
    });
  };
};
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
    return dispatch(userActions.silentUpdate(targetUser, formData))
      .then((user) => {
        dispatch(updateUserSuccess(user));

        return user;
      })
      .catch((response) => {
        const errorObject = formatErrorResponse(response);

        dispatch(updateUserFailure(errorObject));

        throw response;
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

    return Fleet.sessions
      .destroy()
      .then(() => dispatch(logoutSuccess))
      .catch((error) => {
        dispatch(logoutFailure({ base: "Unable to log out of your account" }));

        throw error;
      });
  };
};

export const performRequiredPasswordResetRequest = {
  type: PERFORM_REQUIRED_PASSWORD_RESET_REQUEST,
};

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

    return Fleet.users
      .performRequiredPasswordReset(resetParams)
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

export default {
  ssoRedirect,
  ssoSettings,
};
