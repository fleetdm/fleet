import Kolide from "kolide";

import config from "./config";

const { actions } = config;

export const LOAD_PAGINATED = "LOAD_PAGINATED";

export const loadPaginated = () => {
  const { loadRequest } = actions;
  return (dispatch) => {
    dispatch(loadRequest());

    return Kolide.hosts;
  };
};

export default {
  ...actions,
};
