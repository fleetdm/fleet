import Kolide from '../../../kolide';

export const LOGIN_REQUEST = 'LOGIN_REQUEST';
export const LOGIN_SUCCESS = 'LOGIN_SUCCESS';
export const LOGIN_FAILURE = 'LOGIN_FAILURE';

export const loginRequest = { type: LOGIN_REQUEST };
export const loginSuccess = (user) => {
  return {
    type: LOGIN_SUCCESS,
    payload: {
      data: user,
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

// formData should be { username: <string>, password: <string> }
export const loginUser = (formData) => {
  return (dispatch) => {
    return new Promise((resolve, reject) => {
      dispatch(loginRequest);
      Kolide.loginUser(formData)
        .then(user => {
          dispatch(loginSuccess(user));
          return resolve(user);
        })
        .catch(error => {
          dispatch(loginFailure(error.message));
          return reject(error);
        });
    });
  };
};
