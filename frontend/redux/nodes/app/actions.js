import Kolide from 'kolide';

import formatApiErrors from 'utilities/format_api_errors';
import { frontendFormattedConfig } from 'redux/nodes/app/helpers';

export const CONFIG_FAILURE = 'CONFIG_FAILURE';
export const CONFIG_START = 'CONFIG_START';
export const CONFIG_SUCCESS = 'CONFIG_SUCCESS';
export const SHOW_BACKGROUND_IMAGE = 'SHOW_BACKGROUND_IMAGE';
export const HIDE_BACKGROUND_IMAGE = 'HIDE_BACKGROUND_IMAGE';

export const showBackgroundImage = {
  type: SHOW_BACKGROUND_IMAGE,
};
export const hideBackgroundImage = {
  type: HIDE_BACKGROUND_IMAGE,
};
export const configFailure = (error) => {
  return { type: CONFIG_FAILURE, payload: { error } };
};
export const loadConfig = { type: CONFIG_START };
export const configSuccess = (data) => {
  return { type: CONFIG_SUCCESS, payload: { data } };
};
export const getConfig = () => {
  return (dispatch) => {
    dispatch(loadConfig);

    return Kolide.getConfig()
      .then((config) => {
        const formattedConfig = frontendFormattedConfig(config);

        dispatch(configSuccess(formattedConfig));

        return formattedConfig;
      })
      .catch((error) => {
        const formattedErrors = formatApiErrors(error);

        dispatch(configFailure(formattedErrors));

        throw formattedErrors;
      });
  };
};
export const updateConfig = (configData) => {
  return (dispatch) => {
    dispatch(loadConfig);

    return Kolide.updateConfig(configData)
      .then((config) => {
        const formattedConfig = frontendFormattedConfig(config);

        dispatch(configSuccess(formattedConfig));

        return formattedConfig;
      })
      .catch((error) => {
        const formattedErrors = formatApiErrors(error);

        dispatch(configFailure(formattedErrors));

        throw formattedErrors;
      });
  };
};
