/* eslint-disable no-unused-vars */
import { endsWith, get } from "lodash";
import { push } from "react-router-redux";

import APP_CONSTANTS from "app_constants";
import Fleet from "fleet";
import {
  LOGIN_FAILURE,
  LOGIN_SUCCESS,
  LOGOUT_SUCCESS,
  logoutSuccess,
} from "redux/nodes/auth/actions";
import local from "utilities/local";

const { HTTP_STATUS, PATHS } = APP_CONSTANTS;

const authMiddleware = (store) => (next) => (action) => {
  const { type, payload } = action;

  if (endsWith(type, "FAILURE")) {
    if (get(payload, "errors.http_status") === HTTP_STATUS.UNAUTHENTICATED) {
      store.dispatch(logoutSuccess);
    }
  }

  if (type === LOGIN_SUCCESS) {
    const { token } = payload;

    if (token) {
      local.setItem("auth_token", token);
      Fleet.setBearerToken(token);
    }
  }

  if (type === LOGOUT_SUCCESS || type === LOGIN_FAILURE) {
    const { LOGIN } = PATHS;

    local.removeItem("auth_token");
    Fleet.setBearerToken(null);
    store.dispatch(push(LOGIN));
  }

  return next(action);
};

export default authMiddleware;
