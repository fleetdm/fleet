import Kolide from '../../../../kolide';

export const CLEAR_FORGOT_PASSWORD_ERRORS = 'CLEAR_FORGOT_PASSWORD_ERRORS';
export const FORGOT_PASSWORD_REQUEST = 'FORGOT_PASSWORD_REQUEST';
export const FORGOT_PASSWORD_SUCCESS = 'FORGOT_PASSWORD_SUCCESS';
export const FORGOT_PASSWORD_ERROR = 'FORGOT_PASSWORD_ERROR';

export const clearForgotPasswordErrors = { type: CLEAR_FORGOT_PASSWORD_ERRORS };
export const forgotPasswordRequestAction = { type: FORGOT_PASSWORD_REQUEST };
export const forgotPasswordSuccessAction = (email) => {
  return {
    type: FORGOT_PASSWORD_SUCCESS,
    payload: {
      data: {
        email,
      },
    },
  };
};
export const forgotPasswordErrorAction = (error) => {
  return {
    type: FORGOT_PASSWORD_ERROR,
    payload: {
      error,
    },
  };
};

// formData should be { email: <string> }
export const forgotPasswordAction = (formData) => {
  return (dispatch) => {
    dispatch(forgotPasswordRequestAction);
    return Kolide.forgotPassword(formData)
      .then(() => {
        const { email } = formData;

        return dispatch(forgotPasswordSuccessAction(email));
      })
      .catch(response => {
        const { error } = response;

        dispatch(forgotPasswordErrorAction(error));
        throw response;
      });
  };
};
