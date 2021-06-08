import Fleet from "fleet";

export const VERSION_FAILURE = "VERSION_FAILURE";
export const VERSION_START = "VERSION_START";
export const VERSION_SUCCESS = "VERSION_SUCCESS";

export const loadVersion = { type: VERSION_START };

export const versionSuccess = (data) => {
  return { type: VERSION_SUCCESS, payload: { data } };
};

export const versionFailure = (errors) => {
  return { type: VERSION_FAILURE, payload: { errors } };
};

export const getVersion = () => {
  return (dispatch) => {
    dispatch(loadVersion);

    return Fleet.version
      .load()
      .then((version) => {
        dispatch(versionSuccess(version));

        return version;
      })
      .catch((errors) => {
        dispatch(versionFailure(errors));

        throw errors;
      });
  };
};

export default {
  getVersion,
};
