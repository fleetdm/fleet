import Kolide from "kolide";

import config from "./config";

const { actions } = config;

export const LOAD_PAGINATED = "LOAD_PAGINATED";
export const REFETCH_HOST_START = "REFETCH_HOST_START";
export const REFETCH_HOST_SUCCESS = "REFETCH_HOST_SUCCESS";
export const REFETCH_HOST_FAILURE = "REFETCH_HOST";

export const loadPaginated = () => {
  const { loadRequest } = actions;
  return (dispatch) => {
    dispatch(loadRequest());

    return Kolide.hosts;
  };
};

export const refetchHostSuccess = (data) => {
  return { type: REFETCH_HOST_SUCCESS, payload: { data } };
};

export const refetchHostFailure = (errors) => {
  return { type: REFETCH_HOST_FAILURE, payload: { errors } };
};

export const refetchHostStart = (host) => {
  return (dispatch) => {
    return Kolide.hosts
      .refetch(host)
      .then((data) => {
        dispatch(refetchHostSuccess(data));
        return data;
      })
      .catch((errors) => {
        dispatch(refetchHostFailure(errors));

        throw errors;
      });
  };
};

export default {
  ...actions,
  refetchHostSuccess,
  refetchHostFailure,
  refetchHostStart,
};
