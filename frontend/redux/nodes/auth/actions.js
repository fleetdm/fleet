import md5 from 'js-md5';

import Kolide from '../../../kolide';
import userActions from '../../../redux/nodes/entities/users/actions';

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
export const loginFailure = (error) => {
  return {
    type: LOGIN_FAILURE,
    payload: {
      error,
    },
  };
};

export const fetchCurrentUser = () => {
  return (dispatch) => {
    dispatch(loginRequest);
    return Kolide.me()
      .then((response) => {
        const { user } = response;
        const { email } = user;
        const emailHash = md5(email.toLowerCase());

        user.gravatarURL = `https://www.gravatar.com/avatar/${emailHash}`;
        return dispatch(loginSuccess({ user }));
      })
      .catch((response) => {
        dispatch(loginFailure('Unable to authenticate the current user'));
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
          const { user } = response;
          const { email } = user;
          const emailHash = md5(email.toLowerCase());

          user.gravatarURL = `https://www.gravatar.com/avatar/${emailHash}`;
          dispatch(loginSuccess({ ...response, user }));
          return resolve(user);
        })
        .catch((response) => {
          const { error } = response;
          dispatch(loginFailure(error));
          return reject(error);
        });
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
export const updateUserFailure = (error) => {
  return {
    type: UPDATE_USER_FAILURE,
    payload: {
      error,
    },
  };
};
export const updateUser = (targetUser, formData) => {
  return (dispatch) => {
    return new Promise((resolve, reject) => {
      dispatch(updateUserRequest);
      return dispatch(userActions.update(targetUser, formData))
        .then((response) => {
          const { user } = response;
          dispatch(updateUserSuccess(user));
          return resolve(user);
        })
        .catch((response) => {
          const { error } = response;
          dispatch(updateUserFailure(error));
          return reject(error);
        });
    });
  };
};

export const logoutFailure = (error) => {
  return {
    type: LOGOUT_FAILURE,
    payload: {
      error,
    },
  };
};
export const logoutRequest = { type: LOGOUT_REQUEST };
export const logoutSuccess = { type: LOGOUT_SUCCESS };
export const logoutUser = () => {
  return (dispatch) => {
    dispatch(logoutRequest);
    return Kolide.logout()
      .then(() => {
        return dispatch(logoutSuccess);
      })
      .catch((error) => {
        dispatch(logoutFailure('Unable to log out of your account'));
        throw error;
      });
  };
};
