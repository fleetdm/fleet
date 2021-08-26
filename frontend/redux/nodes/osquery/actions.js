import yaml from "js-yaml";
import Fleet from "fleet";

export const OSQUERY_OPTIONS_FAILURE = "OSQUERY_OPTIONS_FAILURE";
export const OSQUERY_OPTIONS_START = "OSQUERY_OPTIONS_START";
export const OSQUERY_OPTIONS_SUCCESS = "OSQUERY_OPTIONS_SUCCESS";

export const loadOsqueryOptions = { type: OSQUERY_OPTIONS_START };

export const osqueryOptionsSuccess = (data) => {
  return { type: OSQUERY_OPTIONS_SUCCESS, payload: { data } };
};

export const osqueryOptionsFailure = (errors) => {
  return { type: OSQUERY_OPTIONS_FAILURE, payload: { errors } };
};

export const updateOsqueryOptions = (osqueryOptionsData, endpoint) => {
  return (dispatch) => {
    dispatch(loadOsqueryOptions);
    return Fleet.osqueryOptions
      .update(osqueryOptionsData, endpoint)
      .then((osqueryOptions) => {
        const yamlOptions = yaml.safeLoad(osqueryOptionsData.osquery_options);
        dispatch(osqueryOptionsSuccess(yamlOptions));

        return osqueryOptions;
      })
      .catch((errors) => {
        dispatch(osqueryOptionsFailure(errors));

        throw errors;
      });
  };
};

export default {
  updateOsqueryOptions,
};
