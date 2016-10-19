/* eslint-disable no-unused-vars */
import { push } from 'react-router-redux';

import kolide from '../../kolide';
import { LOGIN_FAILURE, LOGIN_SUCCESS, LOGOUT_SUCCESS } from '../nodes/auth/actions';
import local from '../../utilities/local';
import paths from '../../router/paths';

const authMiddleware = store => next => action => {
  const { type, payload } = action;

  if (type === LOGIN_SUCCESS) {
    const { token } = payload;

    if (token) {
      local.setItem('auth_token', token);
      kolide.setBearerToken(token);
    }
  }

  if (type === LOGOUT_SUCCESS || type === LOGIN_FAILURE) {
    const { LOGIN } = paths;

    local.clear();
    kolide.setBearerToken(null);
    store.dispatch(push(LOGIN));
  }

  return next(action);
};

export default authMiddleware;
