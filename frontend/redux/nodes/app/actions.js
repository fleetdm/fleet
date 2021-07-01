import Fleet from "fleet";
import yaml from "js-yaml";

import formatApiErrors from "utilities/format_api_errors";
import { frontendFormattedConfig } from "fleet/helpers";

export const CONFIG_FAILURE = "CONFIG_FAILURE";
export const CONFIG_START = "CONFIG_START";
export const CONFIG_SUCCESS = "CONFIG_SUCCESS";
export const ENROLL_SECRET_FAILURE = "ENROLL_SECRET_FAILURE";
export const ENROLL_SECRET_START = "ENROLL_SECRET_START";
export const ENROLL_SECRET_SUCCESS = "ENROLL_SECRET_SUCCESS";
export const SHOW_BACKGROUND_IMAGE = "SHOW_BACKGROUND_IMAGE";
export const HIDE_BACKGROUND_IMAGE = "HIDE_BACKGROUND_IMAGE";

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
export const enrollSecretFailure = (error) => {
  return { type: ENROLL_SECRET_FAILURE, payload: { error } };
};
export const loadEnrollSecret = { type: ENROLL_SECRET_START };
export const enrollSecretSuccess = (data) => {
  return { type: ENROLL_SECRET_SUCCESS, payload: { data } };
};
export const getConfig = () => {
  return (dispatch) => {
    dispatch(loadConfig);

    return Fleet.config
      .loadAll()
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

    return Fleet.config
      .update(configData)
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
export const getEnrollSecret = () => {
  return (dispatch) => {
    dispatch(loadEnrollSecret);

    return Fleet.config
      .loadEnrollSecret()
      .then((secret) => {
        dispatch(enrollSecretSuccess(secret.spec.secrets));

        return secret;
      })
      .catch((error) => {
        const formattedErrors = formatApiErrors(error);

        dispatch(enrollSecretFailure(formattedErrors));

        throw formattedErrors;
      });
  };
};
