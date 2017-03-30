import Kolide from 'kolide';
import config from 'redux/nodes/entities/config_options/config';
import { formatErrorResponse } from 'redux/nodes/entities/base/helpers';

const { actions } = config;

export const RESET_OPTIONS_START = 'RESET_OPTIONS_START';
export const RESET_OPTIONS_SUCCESS = 'RESET_OPTIONS_SUCCESS';
export const RESET_OPTIONS_FAILURE = 'RESET_OPTIONS_FAILURE';

export const resetOptionsStart = { type: RESET_OPTIONS_START };
export const resetOptionsSuccess = (configOptions) => {
  return { type: RESET_OPTIONS_SUCCESS, payload: { configOptions } };
};
export const resetOptionsFailure = (errors) => {
  return { type: RESET_OPTIONS_FAILURE, payload: { errors } };
};

export const resetOptions = () => {
  return (dispatch) => {
    dispatch(resetOptionsStart);
    return Kolide.configOptions.reset()
       .then((opts) => {
         return dispatch(resetOptionsSuccess(opts));
       })
       .catch((error) => {
         const formattedErrors = formatErrorResponse(error);
         dispatch(resetOptionsFailure(formattedErrors));
         throw formattedErrors;
       });
  };
};

export default {
  ...actions,
  resetOptions,
};
