import Kolide from '../../../kolide';

export const CONFIG_FAILURE = 'CONFIG_FAILURE';
export const CONFIG_START = 'CONFIG_START';
export const CONFIG_SUCCESS = 'CONFIG_SUCCESS';
export const SHOW_BACKGROUND_IMAGE = 'SHOW_BACKGROUND_IMAGE';
export const HIDE_BACKGROUND_IMAGE = 'HIDE_BACKGROUND_IMAGE';
export const SHOW_RIGHT_SIDE_PANEL = 'SHOW_RIGHT_SIDE_PANEL';
export const REMOVE_RIGHT_SIDE_PANEL = 'REMOVE_RIGHT_SIDE_PANEL';

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
      .then(config => {
        dispatch(configSuccess(config));

        return config;
      })
      .catch(error => {
        dispatch(configFailure(error));

        return false;
      });
  };
};
export const showRightSidePanel = {
  type: SHOW_RIGHT_SIDE_PANEL,
};
export const removeRightSidePanel = {
  type: REMOVE_RIGHT_SIDE_PANEL,
};
